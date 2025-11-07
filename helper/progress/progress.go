package progress

import (
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/prettyfmt"
	"github.com/schollz/progressbar/v3"
)

var bar *progressbar.ProgressBar

// Create progress bar once
func InitProgressBar() {
	if bar == nil {
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetElapsedTime(false),
			progressbar.OptionSpinnerType(14), // nolint: mnd
			progressbar.OptionClearOnFinish(),
		)
	}
}

func NonDeterministicFunc(description string, function func() error) error {
	bar.Reset()
	bar.Describe(description)
	bar.Add(1)

	err := function()
	bar.Finish()

	if err != nil {
		prettyfmt.PrettyFmt(" ", prettyfmt.FontRed(prettyfmt.IconCheckFail), " ", prettyfmt.FontWhite(description))
		return err
	}

	prettyfmt.PrettyFmt(" ", prettyfmt.FontGreen(prettyfmt.IconCheckPass), " ", prettyfmt.FontWhite(description))
	return nil
}
