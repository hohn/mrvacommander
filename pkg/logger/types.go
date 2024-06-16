package logger

type LoggerSingle struct {
	modules *Visibles
}

func NewLoggerSingle(v *Visibles) *LoggerSingle {
	l := LoggerSingle{}

	l.modules = v
	return &l
}

type Visibles struct{}
