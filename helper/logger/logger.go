package logger

import (
	"io"
	"log/slog"
	"os"
)

func InitLogger(writer io.Writer, debug bool) {
	handlerOptions := &slog.HandlerOptions{
		AddSource: debug,
	}

	if debug {
		handlerOptions.Level = slog.LevelDebug
	}

	logger := slog.New(customLogHandler(writer, debug, handlerOptions))
	slog.SetDefault(logger)
}

func customLogHandler(writer io.Writer, debug bool, handlerOptions *slog.HandlerOptions) slog.Handler {
	if writer == nil {
		writer = os.Stderr
	}

	if debug {
		return slog.NewJSONHandler(writer, handlerOptions)
	}

	return slog.NewTextHandler(writer, handlerOptions)
}
