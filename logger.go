package smoggytexas

import (
	"log/slog"
	"os"
)

func SetDefaultLogger(level slog.Level) {
	logLevel := &slog.LevelVar{} // INFO
	logLevel.Set(level)
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
	handler1 := slog.NewTextHandler(os.Stderr, &opts)

	slog.SetDefault(slog.New(handler1))
}
