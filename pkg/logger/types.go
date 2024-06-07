package logger

type LoggerSingle struct {
	modules *LoggerVisibles
}

func NewLoggerSingle() *LoggerSingle {
	l := LoggerSingle{}
	return &l
}

type LoggerVisibles struct{}

func (l *LoggerSingle) Setup(v *LoggerVisibles) {
	l.modules = v
}
