package log

var ilog *InnerLog

type InnerLog struct {
	Infof  func(template string, args ...interface{})
	Debugf func(template string, args ...interface{})
	Warnf  func(template string, args ...interface{})
	Errorf func(template string, args ...interface{})
}

func InitInnerLog(i *InnerLog) {
	ilog = &InnerLog{
		Infof:  i.Infof,
		Debugf: i.Debugf,
		Warnf:  i.Warnf,
		Errorf: i.Errorf,
	}
}

func Infof(template string, args ...interface{}) {
	if ilog != nil && ilog.Infof != nil {
		ilog.Infof(template, args...)
	}
}

func Debugf(template string, args ...interface{}) {
	if ilog != nil && ilog.Debugf != nil {
		ilog.Debugf(template, args...)
	}
}

func Warnf(template string, args ...interface{}) {
	if ilog != nil && ilog.Warnf != nil {
		ilog.Warnf(template, args...)
	}
}

func Errorf(template string, args ...interface{}) {
	if ilog != nil && ilog.Errorf != nil {
		ilog.Errorf(template, args...)
	}
}
