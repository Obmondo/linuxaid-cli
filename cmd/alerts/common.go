package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
)

const (
	AnsiReset   = "\033[0m"
	AnsiBold    = "\033[1m"
	AnsiFgGreen = "\033[32m"
	AnsiFgBlue  = "\033[34m"
)

const (
	maxStringInputLength         = 100
	minNumInputValue             = 1
	maxNumInputValue             = 1000
	sleepTimeBetweenInstructions = 500 * time.Millisecond
)

func commandPrint(message string) {
	fmt.Println(fmt.Sprintf("\n%s%s%s\n", AnsiBold+AnsiFgGreen, message, AnsiReset))
}

func headingPrint(message string) {
	fmt.Println(fmt.Sprintf("\n%s%s%s\n", AnsiBold+AnsiFgBlue, message, AnsiReset))
}

func promptForConfirmation(promptText string) bool {
	prompt := promptui.Prompt{
		Label:     promptText,
		IsConfirm: true,
		Validate:  validateConfirmationPrompt,
	}

	_, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Printf("Prompt interrupted %v\n", err)
			os.Exit(0)
		}

	}

	return err == nil
}

func promptForStringInput(promptText string) string {
	prompt := promptui.Prompt{
		Label:    promptText,
		Validate: validateStringInputPrompt,
	}
	result, err := prompt.Run()
	if err != nil {
		handlePromptInputErr(err)
	}

	return strings.TrimSpace(result)
}

func promptForNumInput(promptText string) string {
	prompt := promptui.Prompt{
		Label:    promptText,
		Validate: validateNumInputPrompt,
	}
	result, err := prompt.Run()
	if err != nil {
		handlePromptInputErr(err)
	}

	return strings.TrimSpace(result)
}

func validateConfirmationPrompt(input string) error {
	lowerCaseInput := strings.ToLower(input)
	if lowerCaseInput != "y" && lowerCaseInput != "n" {
		return errors.New("Input must be y or n")
	}

	return nil

}

func validateStringInputPrompt(input string) error {
	if len(input) == 0 {
		return errors.New("Input must not be empty")
	}

	if len(input) > maxStringInputLength {
		return fmt.Errorf("Input must not longer than %d", maxStringInputLength)
	}

	return nil

}

func validateNumInputPrompt(input string) error {
	if len(input) == 0 {
		return errors.New("Input must not be empty")
	}

	val, err := strconv.Atoi(input)
	if err != nil {
		return errors.New("Input must be a number")
	}
	if val < minNumInputValue || val > maxNumInputValue {
		return fmt.Errorf("Input must be in the range %d to %d", minNumInputValue, maxNumInputValue)
	}

	return nil

}

func handlePromptInputErr(err error) {
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			fmt.Printf("Prompt interrupted %v\n", err)
			os.Exit(0)
		}
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}
}

func sleepBetweenInstructions() {
	time.Sleep(sleepTimeBetweenInstructions)
}
