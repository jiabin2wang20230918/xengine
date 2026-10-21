package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var defaultT = TestLog{}

var defaultLog = &InnerLog{
	Infof:  defaultT.Infof,
	Debugf: defaultT.Debugf,
	Warnf:  defaultT.Warnf,
	Errorf: defaultT.Errorf,
}

type TestLog struct {
}

func GetLog() *InnerLog {
	return defaultLog
}

func (t *TestLog) Infof(template string, args ...interface{}) {
	t.Output(3, fmt.Sprintf("\033[32;1m"+template+"\033[0m\n", args...))
}

func (t *TestLog) Debugf(template string, args ...interface{}) {
	t.Output(3, fmt.Sprintf("\033[32;1m"+template+"\033[0m\n", args...))
}

func (t *TestLog) Warnf(template string, args ...interface{}) {
	t.Output(3, fmt.Sprintf("\033[32;1m"+template+"\033[0m\n", args...))
}

func (t *TestLog) Errorf(template string, args ...interface{}) {
	t.Output(3, fmt.Sprintf("\033[32;1m"+template+"\033[0m\n", args...))
}

func (t *TestLog) Output(calldepth int, s string) {
	now := time.Now().Format("01/02 15:04:05.000")
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file) //strings.Split(file, "danbay-sass/")[1]
	}

	logBuf := fmt.Sprintf("%s %s:%d %s", now, file, line, s)
	_, _ = os.Stdout.Write([]byte(logBuf))

}
