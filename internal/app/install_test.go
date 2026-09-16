package app

import (
	"os"
	"testing"
)

func TestShouldContinueAfterConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "yes", input: "yes\n", expected: true},
		{name: "y in any case", input: "Y\n", expected: true},
		{name: "anything else is no", input: "nope\n", expected: false},
		{name: "blank lines ask again", input: "\n  \nyes\n", expected: true},
		{name: "a closed stdin is no instead of asking forever", input: "", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatalf("could not create a pipe: %v", err)
			}

			if _, err := writer.WriteString(test.input); err != nil {
				t.Fatalf("could not write the input: %v", err)
			}

			if err := writer.Close(); err != nil {
				t.Fatalf("could not close the input: %v", err)
			}

			stdin := os.Stdin
			os.Stdin = reader
			t.Cleanup(func() {
				os.Stdin = stdin

				if err := reader.Close(); err != nil {
					t.Errorf("could not close the pipe: %v", err)
				}
			})

			if got := shouldContinueAfterConfirmation(); got != test.expected {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}
