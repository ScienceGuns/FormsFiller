/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bufio"
	"errors"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"strings"
	"testing"
)

// TestSelectForm validates that SelectForm returns the 5320.23 when the settings have a trust
func TestSelectForm(t *testing.T) {
	// Dr. Krieger's settings with his trust filled in
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"

	reader := bufio.NewReader(strings.NewReader("1\n"))
	selectedForm, err := SelectForm(reader, &settings)
	if err != nil {
		t.Fatalf("SelectForm() returned an unexpected error: %v", err)
	}
	if selectedForm.Name != "5320.23 - Responsible Persons Questionnaire" {
		t.Errorf("Expected form to be '%s', but got '%s'", "5320.23 - Responsible Persons Questionnaire", selectedForm.Name)
	}
}

// TestSelectForm_NoTrust validates that SelectForm fails without asking when no form is available
func TestSelectForm_NoTrust(t *testing.T) {
	// Dr. Krieger's settings without a trust
	var settings internal_funcs.FFConfig

	// No input is given, so if SelectForm prompts it will fail on the input instead
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := SelectForm(reader, &settings)
	if err == nil {
		t.Fatalf("Expected SelectForm() to return an error, but got nil")
	}
	if !strings.Contains(err.Error(), "trust-name is blank") {
		t.Errorf("Expected error to mention '%s', but got '%s'", "trust-name is blank", err.Error())
	}
}

// TestSelectForm_SkipsUnavailable validates that unavailable forms are left off the menu and choices map to the right form
func TestSelectForm_SkipsUnavailable(t *testing.T) {
	// Swap in a set of fake forms where the middle one is unavailable
	realSupportedForms := supportedForms
	supportedForms = []ATFForm{
		{Name: "Form 934TXS", CheckSettings: func(settings *internal_funcs.FFConfig) error { return nil }},
		{Name: "Form Danger Zone", CheckSettings: func(settings *internal_funcs.FFConfig) error { return errors.New("lana") }},
		{Name: "Form Phrasing", CheckSettings: func(settings *internal_funcs.FFConfig) error { return nil }},
	}
	defer func() { supportedForms = realSupportedForms }()

	// Option 2 on the menu should be the third form since the second one is skipped
	var settings internal_funcs.FFConfig
	reader := bufio.NewReader(strings.NewReader("2\n"))
	selectedForm, err := SelectForm(reader, &settings)
	if err != nil {
		t.Fatalf("SelectForm() returned an unexpected error: %v", err)
	}
	if selectedForm.Name != "Form Phrasing" {
		t.Errorf("Expected form to be '%s', but got '%s'", "Form Phrasing", selectedForm.Name)
	}
}
