package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	zerolog.Logger
}

func New(level string, pretty bool) Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	zerolog.TimeFieldFormat = time.RFC3339

	var out io.Writer = os.Stdout
	if pretty {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	base := zerolog.New(out).With().Timestamp().Logger()
	return Logger{base}
}

func (l Logger) With(key, value string) Logger {
	return Logger{l.Logger.With().Str(key, value).Logger()}
}
