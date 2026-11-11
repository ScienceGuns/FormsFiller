/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bufio"
	"fmt"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"strings"
)

// SelectForm Ask the user which form to fill out, offering only the forms their settings support
func SelectForm(reader *bufio.Reader, settings *internal_funcs.FFConfig) (*ATFForm, error) {
	// Sort the forms into ones the settings can fill out and ones they can't
	var availableForms []ATFForm
	var unavailableReasons []string
	for _, atfForm := range supportedForms {
		// A form the settings don't support is just left off the menu since not every form applies to everyone
		// (a 5320.23 is only for trusts and legal entities), so the reason is only shown if nothing is left
		if err := atfForm.CheckSettings(settings); err != nil {
			unavailableReasons = append(unavailableReasons, fmt.Sprintf("%s (%v)", atfForm.Name, err))
			continue
		}
		availableForms = append(availableForms, atfForm)
	}
	// There's nothing to pick from if the settings don't support any form
	if len(availableForms) == 0 {
		return nil, fmt.Errorf("failed to find a form the settings file supports: %s", strings.Join(unavailableReasons, "; "))
	}

	// Ask which of the available forms to fill out
	formNames := make([]string, len(availableForms))
	for index, atfForm := range availableForms {
		formNames[index] = atfForm.Name
	}
	choice, err := internal_funcs.PromptChoice(reader, "Select a form:", formNames)
	if err != nil {
		return nil, fmt.Errorf("failed to select a form: %w", err)
	}
	return &availableForms[choice], nil
}
