package logger

type LoggerSingle struct {
	modules *Visibles
}

func NewLoggerSingle() *LoggerSingle {
	l := LoggerSingle{}
	return &l
}

type Visibles struct{}

func (l *LoggerSingle) Setup(v *Visibles) {
	l.modules = v
}
