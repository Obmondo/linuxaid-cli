package progress

import (
	"io"
	"os"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/prettyfmt"
	"github.com/schollz/progressbar/v3"
)

var bar *progressbar.ProgressBar

type (
	indentWriter struct {
		w      io.Writer
		indent string
	}

	progressBarWriter struct {
		bar *progressbar.ProgressBar
	}
)

func (iw *indentWriter) Write(p []byte) (n int, err error) {
	// Replace \r with \r + indent to add indentation after carriage return
	indented := strings.ReplaceAll(string(p), "\r", "\r"+iw.indent)
	return iw.w.Write([]byte(indented))
}

func (pbw *progressBarWriter) Write(p []byte) (n int, err error) {
	return progressbar.Bprintf(pbw.bar, "%s", string(p))
}

// Create progress bar once
func InitProgressBar() *progressBarWriter {
	if bar == nil {
		// Create indented writer
		writer := &indentWriter{
			w:      os.Stdout,
			indent: "  ", // 2 spaces
		}

		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetElapsedTime(false),
			progressbar.OptionSpinnerType(14), // nolint: mnd
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetWriter(writer),
		)
	}

	return &progressBarWriter{
		bar: bar,
	}
}

func NonDeterministicFunc(description string, function func() error) error {
	bar.Reset()
	bar.Describe(description)
	bar.RenderBlank()

	err := function()
	bar.RenderBlank() // nolint: errcheck
	bar.Finish()      // nolint: errcheck

	if err != nil {
		prettyfmt.PrettyPrintf("%s %s\n", prettyfmt.FontRed(prettyfmt.IconCheckFail), prettyfmt.FontWhite(description))
		return err
	}

	prettyfmt.PrettyPrintf("%s %s\n", prettyfmt.FontGreen(prettyfmt.IconCheckPass), prettyfmt.FontWhite(description))
	return nil
}
