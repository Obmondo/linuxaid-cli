package logger

import (
	"log/slog"
	"os"
)

func InitLogger(debug bool) {
	enableSource := true
	handlerOptions := &slog.HandlerOptions{
		AddSource: enableSource,
	}
	if debug {
		loggingLevel := &slog.LevelVar{}
		loggingLevel.Set(slog.LevelDebug)
		handlerOptions.Level = loggingLevel
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, handlerOptions))
	slog.SetDefault(logger)
}
