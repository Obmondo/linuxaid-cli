package logger

import (
	"log/slog"
	"os"
)

func InitLogger(debug bool) {
	handlerOptions := &slog.HandlerOptions{
		AddSource: true,
	}

	if debug {
		handlerOptions.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, handlerOptions))
	slog.SetDefault(logger)
}
