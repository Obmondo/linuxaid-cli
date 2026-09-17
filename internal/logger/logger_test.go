package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestInitLoggerAddsSourceOnlyWhenDebugging pins that log lines carry the source file:line in
// debug mode only.
func TestInitLoggerAddsSourceOnlyWhenDebugging(t *testing.T) {
	defer slog.SetDefault(slog.Default())

	for _, debug := range []bool{false, true} {
		var buf bytes.Buffer
		InitLogger(&buf, debug)
		slog.Info("hello")

		if got := strings.Contains(buf.String(), "source"); got != debug {
			t.Errorf("debug=%t: output %q contains source = %t, want %t", debug, buf.String(), got, debug)
		}
	}
}
