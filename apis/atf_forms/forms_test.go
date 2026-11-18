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

// TestSelectForm validates that SelectForm offers both forms when the settings have a trust
func TestSelectForm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantForm string
	}{
		{"5320.23", "1\n", "5320.23 - Responsible Persons Questionnaire"},
		{"5320.20", "2\n", "5320.20 - Application to Transport Interstate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Dr. Krieger's settings with his trust filled in
			var settings internal_funcs.FFConfig
			settings.UserData.TrustName = "The Krieger Clone Trust"
			settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"

			reader := bufio.NewReader(strings.NewReader(test.input))
			selectedForm, err := SelectForm(reader, &settings)
			if err != nil {
				t.Fatalf("SelectForm() returned an unexpected error: %v", err)
			}
			if selectedForm.Name != test.wantForm {
				t.Errorf("Expected form to be '%s', but got '%s'", test.wantForm, selectedForm.Name)
			}
		})
	}
}

// TestSelectForm_NoTrust validates that without a trust only the 5320.20 is offered
func TestSelectForm_NoTrust(t *testing.T) {
	// Dr. Krieger's settings without a trust
	var settings internal_funcs.FFConfig

	// Option 2 doesn't exist without a trust, so it's rejected before option 1 is accepted
	reader := bufio.NewReader(strings.NewReader("2\n1\n"))
	selectedForm, err := SelectForm(reader, &settings)
	if err != nil {
		t.Fatalf("SelectForm() returned an unexpected error: %v", err)
	}
	if selectedForm.Name != "5320.20 - Application to Transport Interstate" {
		t.Errorf("Expected form to be '%s', but got '%s'", "5320.20 - Application to Transport Interstate", selectedForm.Name)
	}
}

// TestSelectForm_NoneAvailable validates that SelectForm fails without asking, and says why, when no form is available
func TestSelectForm_NoneAvailable(t *testing.T) {
	// Swap in a fake form the settings never support
	realSupportedForms := supportedForms
	supportedForms = []ATFForm{
		{Name: "Form Danger Zone", CheckSettings: func(settings *internal_funcs.FFConfig) error { return errors.New("lana") }},
	}
	defer func() { supportedForms = realSupportedForms }()

	// No input is given, so if SelectForm prompts it will fail on the input instead
	var settings internal_funcs.FFConfig
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := SelectForm(reader, &settings)
	if err == nil {
		t.Fatalf("Expected SelectForm() to return an error, but got nil")
	}
	if !strings.Contains(err.Error(), "Form Danger Zone (lana)") {
		t.Errorf("Expected error to mention '%s', but got '%s'", "Form Danger Zone (lana)", err.Error())
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
