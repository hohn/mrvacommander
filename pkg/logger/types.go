package logger

type LoggerSingle struct {
}

func NewLoggerSingle() *LoggerSingle {
	l := LoggerSingle{}
	return &l
}
