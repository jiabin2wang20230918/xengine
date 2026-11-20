package log

import "sync"

var (
	ilogMu sync.RWMutex
	ilog   *InnerLog
)

type InnerLog struct {
	Infof  func(template string, args ...interface{})
	Debugf func(template string, args ...interface{})
	Warnf  func(template string, args ...interface{})
	Errorf func(template string, args ...interface{})
}

func InitInnerLog(i *InnerLog) {
	ilogMu.Lock()
	defer ilogMu.Unlock()
	ilog = &InnerLog{
		Infof:  i.Infof,
		Debugf: i.Debugf,
		Warnf:  i.Warnf,
		Errorf: i.Errorf,
	}
}

func Infof(template string, args ...interface{}) {
	ilogMu.RLock()
	l := ilog
	ilogMu.RUnlock()
	if l != nil && l.Infof != nil {
		l.Infof(template, args...)
	}
}

func Debugf(template string, args ...interface{}) {
	ilogMu.RLock()
	l := ilog
	ilogMu.RUnlock()
	if l != nil && l.Debugf != nil {
		l.Debugf(template, args...)
	}
}

func Warnf(template string, args ...interface{}) {
	ilogMu.RLock()
	l := ilog
	ilogMu.RUnlock()
	if l != nil && l.Warnf != nil {
		l.Warnf(template, args...)
	}
}

func Errorf(template string, args ...interface{}) {
	ilogMu.RLock()
	l := ilog
	ilogMu.RUnlock()
	if l != nil && l.Errorf != nil {
		l.Errorf(template, args...)
	}
}
