package main

import (
	"golang.org/x/exp/slog"
	"os"
)

func setupLogging(logger *slog.Logger, verbose, veryVerbose bool) {
	logLevel := &slog.LevelVar{} // INFO
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

	if veryVerbose {
		opts.Level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(handler1))

	opts2 := opts
	if verbose {
		opts2.Level = slog.LevelDebug
	}

	handler2 := slog.NewTextHandler(os.Stderr, &opts2)

	*logger = slog.New(handler2)
}
