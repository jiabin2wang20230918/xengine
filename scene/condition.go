package scene

import (
	"github.com/jiabin2wang20230918/xengine/compare"
	"github.com/jiabin2wang20230918/xengine/shadow"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"strconv"
	"strings"
	"time"
)

// Condition 条件需要实现自校验功能
type Condition interface {
	// IsMatch 检查自己是否匹配
	IsMatch() bool
}

// PropertyCondition 属性条件，提供属性相关检测功能
type PropertyCondition struct {
	// deviceId 设备id
	deviceId string
	// propertyName 属性名称
	propertyName string
	// propertyValue 属性值
	propertyValue interface{}
	// symbol 符号 例如 ==,>=
	symbol string
}

// Match 主动和别的值进行对比是否匹配，可以用于trigger
func (p *PropertyCondition) Match(propertyName string, propertyValue interface{}) bool {
	if p.propertyName == propertyName {
		return compare.PropertyMatch(propertyValue, p.propertyValue, p.symbol)
	} else {
		return false
	}
}

// IsMatch 自检测条件是否匹配，基于 shadow 模块检测。
func (p PropertyCondition) IsMatch() bool {
	v, ok := shadow.GetPropertyShadow(p.deviceId, p.propertyName)
	if !ok {
		return false
	}
	return compare.PropertyMatch(v, p.propertyValue, p.symbol)
}

// EventCondition 事件条件，提供事件相关检测功能
type EventCondition struct {
	eventName string
}

// Match 主动检测是否匹配，可以用于trigger
func (p *EventCondition) Match(eventName string) bool {
	return p.eventName == eventName
}

// TimeCondition 时间条件，负责对时间条件进行响应。
type TimeCondition struct {
	startTime string
	endTime   string
	time      string
}

// IsMatch
// 时间条件 是否匹配，自检测是否匹配
func (c *TimeCondition) IsMatch() bool {
	// 检查起止时间
	if c.startEndTimeMatch(time.Now()) {
		// 检查cron
		return cronMatchNow(c.time)
	} else {
		return false
	}
}

// startEndTimeMatch 对比起始时间是否匹配
// 任何异常（格式异常，类型异常）都会返回false
// 08:59 09:01 --> 09:00 true
func (c *TimeCondition) startEndTimeMatch(compareTime time.Time) bool {
	startTime := strings.Split(c.startTime, ":")
	endTime := strings.Split(c.endTime, ":")
	if len(startTime) < 2 {
		log.Errorf("startTime(%s) length not legal", c.startTime)
		return false
	}
	if len(endTime) < 2 {
		log.Errorf("endTime(%s) length not legal", c.endTime)
		return false
	}
	startTimeHour, err := strconv.ParseInt(startTime[0], 10, 32)
	if err != nil {
		log.Errorf("startTimeHour(%s) parse int error:%v", startTime[0], err)
		return false
	}
	startTimeMinute, err := strconv.ParseInt(startTime[1], 10, 32)
	if err != nil {
		log.Errorf("startTimeMinute(%s) parse int error:%v", startTime[1], err)
		return false
	}
	endTimeHour, err := strconv.ParseInt(endTime[0], 10, 32)
	if err != nil {
		log.Errorf("endTimeHour(%s) parse int error:%v", endTime[0], err)
		return false
	}
	endTimeMinute, err := strconv.ParseInt(endTime[1], 10, 32)
	if err != nil {
		log.Errorf("endTimeMinute(%s) parse int error:%v", endTime[1], err)
		return false
	}
	log.Debugf("startTimeHour:%d, startTimeMinute:%d, endTimeHour:%d, endTimeMinute:%d, timeNowHour:%d, timeNowMinute:%d", startTimeHour, startTimeMinute, endTimeHour, endTimeMinute, compareTime.Hour(), compareTime.Minute())

	// start时间判定，要么现在的小时大于起始时间，要么现在的小时等于起始时间同时当前分钟大于起始分钟。
	startFlag := int(startTimeHour) < compareTime.Hour() || (int(startTimeHour) == compareTime.Hour() && int(startTimeMinute) <= compareTime.Minute())
	endFlag := int(endTimeHour) > compareTime.Hour() || (int(endTimeHour) == compareTime.Hour() && int(endTimeMinute) >= compareTime.Minute())

	return startFlag && endFlag
}

// cronMatchNow cron匹配
// 目前仅匹配了日，月，周，年。
// 基于 cronMatchSymbol 方法。
func cronMatchNow(cronData string) bool {
	split := strings.Split(cronData, " ")
	if len(split) < 7 {
		log.Errorf("cron data(%s) length err", cronData)
		return false
	}

	dayString := split[3]
	monthString := split[4]
	weekString := split[5]
	yearString := split[6]

	timeNow := time.Now()

	// 注意当前day of week需+1才和云端协议匹配
	return cronMatchSymbol(timeNow.Day(), dayString) && cronMatchSymbol(int(timeNow.Month()), monthString) && cronMatchSymbol(int(timeNow.Weekday())+1, weekString) && cronMatchSymbol(timeNow.Year(), yearString)
}

// cronMatchSymbol cron符号匹配
// 符号支持 '?' '*' 'number, number'
func cronMatchSymbol(value int, cronSingle string) bool {
	switch cronSingle {
	case "?":
		return true
	case "*":
		return true
	default:
		d := strings.Split(cronSingle, ",")
		for _, s := range d {
			parseInt, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				log.Errorf("cron data(%s) string parse int err:%v", s, err)
				return false
			}
			if value == int(parseInt) {
				// day not match
				log.Debugf("one of cron filed data(%d) match value(%d)", parseInt, value)
				return true
			}
		}
		log.Debugf("cron filed(%v) not match value(%d)", d, value)
		return false
	}
}
