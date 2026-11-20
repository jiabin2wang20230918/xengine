package log

import "sync/atomic"

var ilog atomic.Pointer[InnerLog]

type InnerLog struct {
	Infof  func(template string, args ...interface{})
	Debugf func(template string, args ...interface{})
	Warnf  func(template string, args ...interface{})
	Errorf func(template string, args ...interface{})
}

func InitInnerLog(i *InnerLog) {
	if i == nil {
		ilog.Store(nil)
		return
	}
	cp := *i
	ilog.Store(&cp)
}

func Infof(template string, args ...interface{}) {
	if l := ilog.Load(); l != nil && l.Infof != nil {
		l.Infof(template, args...)
	}
}

func Debugf(template string, args ...interface{}) {
	if l := ilog.Load(); l != nil && l.Debugf != nil {
		l.Debugf(template, args...)
	}
}

func Warnf(template string, args ...interface{}) {
	if l := ilog.Load(); l != nil && l.Warnf != nil {
		l.Warnf(template, args...)
	}
}

func Errorf(template string, args ...interface{}) {
	if l := ilog.Load(); l != nil && l.Errorf != nil {
		l.Errorf(template, args...)
	}
}
