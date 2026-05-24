package logger

type Log struct{}

func New(env string) *Log {
	return new(Log)
}
