package scene

import (
	"strconv"
	"strings"
	"time"

	"github.com/jiabin2wang20230918/xengine/compare"
	"github.com/jiabin2wang20230918/xengine/shadow"
	"github.com/jiabin2wang20230918/xengine/utils/log"
)

type propertyCondition struct {
	deviceId      string
	propertyName  string
	propertyValue interface{}
	symbol        string
}

func (p *propertyCondition) match(propertyName string, propertyValue interface{}) bool {
	if p.propertyName != propertyName {
		return false
	}
	return compare.PropertyMatch(propertyValue, p.propertyValue, p.symbol)
}

func (p *propertyCondition) isMatch() bool {
	v, ok := shadow.GetPropertyShadow(p.deviceId, p.propertyName)
	if !ok {
		return false
	}
	return compare.PropertyMatch(v, p.propertyValue, p.symbol)
}

type eventCondition struct {
	eventName string
}

func (p *eventCondition) match(eventName string) bool {
	return p.eventName == eventName
}

type timeCondition struct {
	startTime string
	endTime   string
	time      string
}

func (c *timeCondition) isMatch() bool {
	return c.isMatchAt(time.Now())
}

func (c *timeCondition) isMatchAt(now time.Time) bool {
	if !c.startEndTimeMatch(now) {
		return false
	}
	return cronMatchTime(c.time, now)
}

func parseHourMinute(raw string) (int, int, bool) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	hour, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, false
	}
	minute, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}

// startEndTimeMatch checks whether compareTime's HH:MM falls inside [startTime, endTime].
// Supports both daytime windows (e.g. 08:59-09:01) and overnight windows (e.g. 22:00-06:00).
func (c *timeCondition) startEndTimeMatch(compareTime time.Time) bool {
	startHour, startMinute, ok := parseHourMinute(c.startTime)
	if !ok {
		log.Errorf("startTime(%s) is invalid", c.startTime)
		return false
	}
	endHour, endMinute, ok := parseHourMinute(c.endTime)
	if !ok {
		log.Errorf("endTime(%s) is invalid", c.endTime)
		return false
	}

	startTotal := startHour*60 + startMinute
	endTotal := endHour*60 + endMinute
	nowTotal := compareTime.Hour()*60 + compareTime.Minute()

	log.Debugf("startTime=%02d:%02d endTime=%02d:%02d now=%02d:%02d",
		startHour, startMinute, endHour, endMinute, compareTime.Hour(), compareTime.Minute())

	if startTotal <= endTotal {
		return nowTotal >= startTotal && nowTotal <= endTotal
	}
	// Overnight window wraps across midnight.
	return nowTotal >= startTotal || nowTotal <= endTotal
}

var (
	monthAliases = map[string]int{
		"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4,
		"MAY": 5, "JUN": 6, "JUL": 7, "AUG": 8,
		"SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
	}
	// Cloud protocol weekday numbering: 1=SUN .. 7=SAT.
	weekdayAliases = map[string]int{
		"SUN": 1, "MON": 2, "TUE": 3, "WED": 4,
		"THU": 5, "FRI": 6, "SAT": 7,
	}
)

// cronMatchTime checks day, month, weekday (1=SUN..7=SAT), and year fields of a 7-field cron string.
func cronMatchTime(cronData string, now time.Time) bool {
	split := strings.Fields(cronData)
	if len(split) < 7 {
		log.Errorf("cron data(%s) length err", cronData)
		return false
	}

	dayString := split[3]
	monthString := split[4]
	weekString := split[5]
	yearString := split[6]

	return cronMatchField(now.Day(), dayString, 1, 31, nil) &&
		cronMatchField(int(now.Month()), monthString, 1, 12, monthAliases) &&
		cronMatchField(int(now.Weekday())+1, weekString, 1, 7, weekdayAliases) &&
		cronMatchField(now.Year(), yearString, 1970, 2099, nil)
}

func parseCronToken(token string, aliases map[string]int) (int, bool) {
	t := strings.TrimSpace(token)
	if t == "" {
		return 0, false
	}
	if aliases != nil {
		if v, ok := aliases[strings.ToUpper(t)]; ok {
			return v, true
		}
	}
	v, err := strconv.Atoi(t)
	if err != nil {
		return 0, false
	}
	return v, true
}

// cronMatchField matches a single cron field supporting '?', '*', lists (a,b), ranges (a-b), and steps (*/n, a-b/n, a/n).
func cronMatchField(value int, field string, minVal, maxVal int, aliases map[string]int) bool {
	field = strings.TrimSpace(field)
	if field == "?" || field == "*" {
		return true
	}

	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if matchCronPart(value, part, minVal, maxVal, aliases) {
			return true
		}
	}
	log.Debugf("cron field(%s) does not match value(%d)", field, value)
	return false
}

func matchCronPart(value int, part string, minVal, maxVal int, aliases map[string]int) bool {
	step := 1
	base := part
	hasStep := false

	if idx := strings.IndexByte(part, '/'); idx >= 0 {
		base = part[:idx]
		stepStr := part[idx+1:]
		s, err := strconv.Atoi(strings.TrimSpace(stepStr))
		if err != nil || s <= 0 {
			log.Errorf("cron step(%s) invalid", part)
			return false
		}
		step = s
		hasStep = true
	}

	if base == "*" || base == "?" {
		return value >= minVal && value <= maxVal && (value-minVal)%step == 0
	}

	if idx := strings.IndexByte(base, '-'); idx >= 0 {
		start, ok1 := parseCronToken(base[:idx], aliases)
		end, ok2 := parseCronToken(base[idx+1:], aliases)
		if !ok1 || !ok2 || start > end {
			log.Errorf("cron range(%s) invalid", part)
			return false
		}
		return value >= start && value <= end && (value-start)%step == 0
	}

	start, ok := parseCronToken(base, aliases)
	if !ok {
		log.Errorf("cron token(%s) invalid", part)
		return false
	}
	if hasStep {
		return value >= start && value <= maxVal && (value-start)%step == 0
	}
	return value == start
}
