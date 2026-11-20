package scene

import (
	"testing"
	"time"
)

func TestTimeCondition_StartEndTimeMatch(t *testing.T) {
	utc := func(h, m int) time.Time {
		return time.Date(2026, time.November, 20, h, m, 0, 0, time.UTC)
	}

	tests := []struct {
		name      string
		startTime string
		endTime   string
		now       time.Time
		want      bool
	}{
		{"full day", "0:00", "23:59", utc(12, 30), true},
		{"daytime inside", "8:59", "09:01", utc(9, 0), true},
		{"daytime start boundary", "8:59", "09:01", utc(8, 59), true},
		{"daytime end boundary", "8:59", "09:01", utc(9, 1), true},
		{"daytime before", "8:59", "09:01", utc(8, 58), false},
		{"daytime after", "8:59", "09:01", utc(9, 2), false},
		{"overnight late evening", "22:00", "06:00", utc(23, 30), true},
		{"overnight midnight", "22:00", "06:00", utc(0, 0), true},
		{"overnight early morning", "22:00", "06:00", utc(5, 59), true},
		{"overnight start boundary", "22:00", "06:00", utc(22, 0), true},
		{"overnight end boundary", "22:00", "06:00", utc(6, 0), true},
		{"overnight outside midday", "22:00", "06:00", utc(12, 0), false},
		{"invalid hour", "25:00", "23:59", utc(12, 0), false},
		{"invalid minute", "08:60", "23:59", utc(12, 0), false},
		{"malformed format", "0800", "23:59", utc(12, 0), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := timeCondition{startTime: tc.startTime, endTime: tc.endTime}
			if got := c.startEndTimeMatch(tc.now); got != tc.want {
				t.Fatalf("startEndTimeMatch(%s-%s @ %v) = %v, want %v", tc.startTime, tc.endTime, tc.now, got, tc.want)
			}
		})
	}
}

func TestCronMatchTime(t *testing.T) {
	// 2026-11-20 is Friday (time.Weekday=5 -> cloud weekday=6 / FRI).
	fri := time.Date(2026, time.November, 20, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		cron string
		now  time.Time
		want bool
	}{
		{"wildcards", "* * * * * ? *", fri, true},
		{"exact day match", "* * * 20 * ? *", fri, true},
		{"exact day mismatch", "* * * 13 * ? *", fri, false},
		{"day list match", "* * * 13,20,25 * ? *", fri, true},
		{"month list match", "* * * 20 1,11 ? *", fri, true},
		{"month name match", "* * * 20 NOV ? *", fri, true},
		{"month range match", "* * * 20 10-12 ? *", fri, true},
		{"weekday numeric match", "* * * ? * 6 *", fri, true},
		{"weekday range match", "* * * ? * 2-6 *", fri, true},
		{"weekday name match", "* * * ? * WED,FRI *", fri, true},
		{"weekday mismatch", "* * * ? * MON,TUE *", fri, false},
		{"day step match", "* * * 10-30/5 * ? *", fri, true},
		{"day step mismatch", "* * * 10-30/3 * ? *", fri, false},
		{"year exact match", "* * * 20 11 6 2026", fri, true},
		{"year range match", "* * * 20 11 6 2020-2030", fri, true},
		{"year mismatch", "* * * 20 11 6 2023", fri, false},
		{"too few fields", "* * * 20 11 6", fri, false},
		{"invalid range", "* * * 25-10 * ? *", fri, false},
		{"invalid step zero", "* * * */0 * ? *", fri, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cronMatchTime(tc.cron, tc.now); got != tc.want {
				t.Fatalf("cronMatchTime(%q, %v) = %v, want %v", tc.cron, tc.now, got, tc.want)
			}
		})
	}
}

func TestTimeCondition_IsMatchAt(t *testing.T) {
	friNight := time.Date(2026, time.November, 20, 23, 15, 0, 0, time.UTC)
	c := timeCondition{
		startTime: "22:00",
		endTime:   "06:00",
		time:      "* * * * * FRI *",
	}
	if !c.isMatchAt(friNight) {
		t.Fatal("expected overnight Friday condition to match")
	}
	if c.isMatchAt(time.Date(2026, time.November, 20, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("expected midday to not match overnight window")
	}
}
