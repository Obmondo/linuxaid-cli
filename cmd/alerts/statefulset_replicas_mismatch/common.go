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
	maxStringInputLength  = 100
	minNumInputValue      = 1
	maxNumInputValue      = 1000
	sleepTimeBetweenSteps = 200 * time.Millisecond
)

var scriptInputsMap = map[string]string{}

var (
	errMoveOn = errors.New("move on")
	errBreak  = errors.New("break")
	errExit   = errors.New("exit")
)

// A script step is the building block of each alert script
// Each script is essentially meant to be a collection of script steps executed one after the other
type scriptStep struct {
	// This is the input passed to the script step
	Message func() string
	// This is the function that will be executed based on the input passed to the step
	ExecuteFunc func(string) string
	// This is the key in the script inputs map for which the step's output will be stored
	StoreFunctionOutputAs string
	// This is the message printed when the step is over
	ExitMessage string
	// If the step returns 'true' or 'false', then an error is returned if the return value matches 'ExitOn'
	// Different types of error can be returned depending on the value of ContinueLoop and BreakLoop
	ExitOn bool
	// If this field is specified, the step runs only if the corresponding key exists in the script inputs map
	RunIfKeyExists string
	// Useful only for loops, return an error that indicates that a loop needs to be continued
	ContinueLoop bool
	// Useful only for loops, return an error that indicates that a loop needs to break
	BreakLoop bool
}

// executeSteps executes a sequence of steps
func executeSteps(steps []scriptStep) error {
	for _, step := range steps {
		time.Sleep(sleepTimeBetweenSteps)
		if len(step.RunIfKeyExists) > 0 && !(scriptInputsMap[step.RunIfKeyExists] == strconv.FormatBool(true)) {
			continue
		}

		result := step.ExecuteFunc(step.Message())
		if result != "" && len(step.StoreFunctionOutputAs) > 0 {
			scriptInputsMap[step.StoreFunctionOutputAs] = result

		}
		resultBool, err := strconv.ParseBool(result)
		if err == nil && resultBool == step.ExitOn && len(step.ExitMessage) > 0 {
			simplePrint(step.ExitMessage)
			switch {

			case step.BreakLoop:
				return errBreak

			case step.ContinueLoop:
				return errMoveOn

			default:
				return errExit
			}
		}
	}
	return nil

}

func commandPrint(message string) string {
	fmt.Println("Run the following command:")
	fmt.Println(fmt.Sprintf("\n%s%s%s\n", AnsiBold+AnsiFgGreen, message, AnsiReset))
	return ""
}

func headingPrint(message string) string {
	fmt.Println(fmt.Sprintf("\n%s%s%s\n", AnsiBold+AnsiFgBlue, message, AnsiReset))
	return ""
}

func simplePrint(message string) string {
	fmt.Println(message)
	return ""
}

func compareStrings(message string) string {

	messageParts := strings.Split(message, ",")
	if len(messageParts) != 2 {
		return strconv.FormatBool(false)
	}

	return strconv.FormatBool(messageParts[0] == scriptInputsMap[messageParts[1]])

}

func assignNewMapKeysFromExistingKeys(message string) string {
	messageParts := strings.Split(message, ",")
	if len(messageParts)%2 != 0 || len(messageParts) < 2 {
		return strconv.FormatBool(false)
	}

	for i := 0; i < len(messageParts); i += 2 {
		scriptInputsMap[messageParts[i]] = scriptInputsMap[messageParts[i+1]]
	}
	return strconv.FormatBool(true)
}

func promptForConfirmation(promptText string) string {
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

	return strconv.FormatBool(err == nil)
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
