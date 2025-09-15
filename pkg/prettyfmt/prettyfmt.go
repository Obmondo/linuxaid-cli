package prettyfmt

import (
	"fmt"

	"github.com/fatih/color"
)

const (
	IconGear     = "⚙️"
	IconCheck    = "✔"
	IconIceCream = "🎉"
)

// nolint: revive
var FontRed, FontGreen, FontBlue, FontWhite, FontYellow func(a ...interface{}) string

func init() {
	FontGreen = color.New(color.FgGreen).SprintFunc()
	FontBlue = color.New(color.FgBlue).SprintFunc()
	FontWhite = color.New(color.FgWhite).SprintFunc()
	FontYellow = color.New(color.FgYellow).SprintFunc()
	FontRed = color.New(color.FgRed).SprintFunc()
}

func PrettyFmt(a ...any) {
	//nolint:forbidigo
	fmt.Println(a...) //nolint:revive
}
