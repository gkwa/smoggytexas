package logging

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func MakeLogger() *slog.Logger {
	logLevel := &slog.LevelVar{} // INFO
	logLevel.Set(slog.LevelInfo)
	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}
	handler1 := slog.NewTextHandler(os.Stdout, &opts)

	// default logger customized
	slog.SetDefault(slog.New(handler1))

	handler2 := slog.NewTextHandler(os.Stderr, &opts)

	logger = slog.New(handler2)
	return logger
}
