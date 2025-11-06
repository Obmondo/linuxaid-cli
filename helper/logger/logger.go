package logger

import (
	"log/slog"
	"os"
)

func InitLogger(debug bool) {
	handlerOptions := &slog.HandlerOptions{
		AddSource: debug,
	}

	if debug {
		handlerOptions.Level = slog.LevelDebug
	}

	logger := slog.New(customLogHandler(debug, handlerOptions))
	slog.SetDefault(logger)
}

func customLogHandler(debug bool, handlerOptions *slog.HandlerOptions) slog.Handler {
	if debug {
		return slog.NewJSONHandler(os.Stderr, handlerOptions)
	}
	return slog.NewTextHandler(os.Stderr, handlerOptions)
}
