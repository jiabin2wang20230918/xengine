package scene

import (
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"testing"
	"time"
)

func TestTimeCondition_IsMatch(t *testing.T) {
	log.InitInnerLog(log.GetLog())
	temp := TimeCondition{
		startTime: "0:00",
		endTime:   "23:59",
		time:      "* * * * * ? *",
	}
	t.Log(temp.IsMatch())

	temp.time = "* * * 13 * ? *"
	t.Log(temp.IsMatch())

	temp.time = "* * * 13,1 * ? *"
	t.Log(temp.IsMatch())

	temp.time = "* * * 13,1,9 * ? *"
	t.Log(temp.IsMatch())

	temp.time = "* * * 13,1,9 1,3,9 ? *"
	t.Log(temp.IsMatch())

	temp.time = "* * * 13,1,9 3,9 ? *"
	t.Log(temp.IsMatch())

	temp.time = "* * * 13,1,12 1,3,9 1,2,3,4,6 *"
	t.Log(temp.IsMatch())

	temp.time = "* * * 13,1,12 1,3,10 1,2,5 2023"
	t.Log(temp.IsMatch())
}

func TestTimeCondition_StartEndTimeMatch(t *testing.T) {
	temp := TimeCondition{
		startTime: "0:00",
		endTime:   "23:59",
	}
	log.InitInnerLog(log.GetLog())
	t.Log(temp.startEndTimeMatch(time.Now()))

	temp.startTime = "8:59"
	temp.endTime = "09:01"

	// 正向
	t.Log(temp.startEndTimeMatch(time.Date(0, 0, 0, 9, 0, 0, 0, time.UTC)))
	t.Log(temp.startEndTimeMatch(time.Date(0, 0, 0, 8, 59, 0, 0, time.UTC)))
	t.Log(temp.startEndTimeMatch(time.Date(0, 0, 0, 9, 1, 0, 0, time.UTC)))

	// 反向
	t.Log(!temp.startEndTimeMatch(time.Date(0, 0, 0, 9, 2, 0, 0, time.UTC)))
	t.Log(!temp.startEndTimeMatch(time.Date(0, 0, 0, 8, 58, 0, 0, time.UTC)))
}

func TestTime(t *testing.T) {
	parse, err := time.Parse("15:04", "3:08")
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}
	t.Log(parse)

	t.Log(parse.Format("15:04"))
	parse, err = time.Parse("15:04", "03:08")
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}
	t.Log(parse)

	t.Log(parse.Format("15:04"))
}
