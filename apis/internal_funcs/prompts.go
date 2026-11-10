/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package internal_funcs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PromptRequiredText Ask a question until the user gives a non-blank answer
func PromptRequiredText(reader *bufio.Reader, question string) (string, error) {
	for {
		fmt.Printf("%s: ", question)
		answer, err := readLine(reader)
		if err != nil {
			return "", err
		}
		if answer != "" {
			return answer, nil
		}
		fmt.Println("An answer is required. Please try again.")
	}
}

// PromptOptionalText Ask a question that may be left blank, but only once the user leaves it blank a second time
func PromptOptionalText(reader *bufio.Reader, question string) (string, error) {
	fmt.Printf("%s: ", question)
	answer, err := readLine(reader)
	if err != nil || answer != "" {
		return answer, err
	}
	// A blank answer could just be a stray press of Enter, so warn and give one more chance
	fmt.Println("Warning: This was left blank. Leave it blank again and it will be left blank on the form.")
	fmt.Printf("%s: ", question)
	return readLine(reader)
}

// PromptChoice Show a numbered list of options and return the index of the one the user picks
func PromptChoice(reader *bufio.Reader, heading string, options []string) (int, error) {
	// A menu with nothing on it can't be answered
	if len(options) == 0 {
		return 0, errors.New("failed to prompt for a choice: no options were given")
	}

	// Display the options numbered from 1 since that's what people expect to type
	fmt.Println(heading)
	for index, option := range options {
		fmt.Printf("  %d: %s\n", index+1, option)
	}
	// Show the valid range, which is just [1] when there's only one option
	validRange := "1"
	if len(options) > 1 {
		validRange = fmt.Sprintf("1-%d", len(options))
	}
	// Only permit a valid selection before continuing
	for {
		fmt.Printf("Enter a number [%s]: ", validRange)
		answer, err := readLine(reader)
		if err != nil {
			return 0, err
		}
		choice, err := strconv.Atoi(answer)
		if err == nil && choice >= 1 && choice <= len(options) {
			return choice - 1, nil
		}
		fmt.Println("Invalid choice. Please try again.")
	}
}

// PromptYesNo Ask a yes or no question, returning defaultAnswer when the user just presses Enter
func PromptYesNo(reader *bufio.Reader, question string, defaultAnswer bool) (bool, error) {
	// Capitalize the default so the user can see what Enter does
	answerHint := "y/N"
	if defaultAnswer {
		answerHint = "Y/n"
	}
	for {
		fmt.Printf("%s (%s): ", question, answerHint)
		answer, err := readLine(reader)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return defaultAnswer, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Println("Please answer y or n.")
	}
}

// readLine Read one line of input with surrounding whitespace removed
func readLine(reader *bufio.Reader) (string, error) {
	input, err := reader.ReadString('\n')
	// A final line without a newline still counts, so only fail when EOF arrives with nothing to use
	if err != nil && (!errors.Is(err, io.EOF) || input == "") {
		if errors.Is(err, io.EOF) {
			return "", errors.New("failed to read input: input ended before an answer was given")
		}
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}
