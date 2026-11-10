/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package internal_funcs

import (
	"bufio"
	"strings"
	"testing"
)

// TestPromptRequiredText validates that PromptRequiredText re-asks on blank answers and trims the result
func TestPromptRequiredText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"First try", "Barry Dylan\n", "Barry Dylan", false},
		{"Blank lines first", "\n   \nBarry Dylan\n", "Barry Dylan", false},
		{"Surrounding whitespace", "  Barry Dylan  \n", "Barry Dylan", false},
		{"Last line without a newline", "Barry Dylan", "Barry Dylan", false},
		{"Input ends with no answer", "\n\n", "", true},
		{"No input at all", "", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.input))
			got, err := PromptRequiredText(reader, "Name of the cyborg")
			if (err != nil) != test.wantErr {
				t.Fatalf("PromptRequiredText() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("Expected answer to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestPromptOptionalText validates that a blank answer is only accepted when it's given twice in a row
func TestPromptOptionalText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"First try", "934 Texas Ave\n", "934 Texas Ave", false},
		{"Blank then an answer", "\n934 Texas Ave\n", "934 Texas Ave", false},
		{"Blank twice", "\n\n", "", false},
		{"Whitespace twice", "  \n \n", "", false},
		{"Input ends after one blank", "\n", "", true},
		{"No input at all", "", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.input))
			got, err := PromptOptionalText(reader, "Address of the Figgis Agency")
			if (err != nil) != test.wantErr {
				t.Fatalf("PromptOptionalText() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("Expected answer to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestPromptChoice validates that PromptChoice only accepts a number from the menu and returns its index
func TestPromptChoice(t *testing.T) {
	options := []string{"Lana Kane", "Sterling Archer", "Cyril Figgis"}
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"First option", "1\n", 0, false},
		{"Last option", "3\n", 2, false},
		{"Out of range then valid", "0\n4\n2\n", 1, false},
		{"Not a number then valid", "Archer\n\n2\n", 1, false},
		{"Input ends with no answer", "9\n", 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.input))
			got, err := PromptChoice(reader, "Who drank Krieger's serum?", options)
			if (err != nil) != test.wantErr {
				t.Fatalf("PromptChoice() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("Expected choice to be '%d', but got '%d'", test.want, got)
			}
		})
	}
}

// TestPromptChoice_NoOptions validates that PromptChoice refuses to show an empty menu
func TestPromptChoice_NoOptions(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("1\n"))
	if _, err := PromptChoice(reader, "Pick a clone", []string{}); err == nil {
		t.Errorf("Expected PromptChoice() to return an error, but got nil")
	}
}

// TestPromptYesNo validates yes and no answers, the default on Enter, and re-asking on anything else
func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		defaultAnswer bool
		want          bool
		wantErr       bool
	}{
		{"y", "y\n", false, true, false},
		{"YES", "YES\n", false, true, false},
		{"n", "n\n", true, false, false},
		{"No", "No\n", true, false, false},
		{"Enter with default yes", "\n", true, true, false},
		{"Enter with default no", "\n", false, false, false},
		{"Nonsense then yes", "phrasing\ny\n", false, true, false},
		{"Input ends with no answer", "phrasing\n", false, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.input))
			got, err := PromptYesNo(reader, "Do you want ants?", test.defaultAnswer)
			if (err != nil) != test.wantErr {
				t.Fatalf("PromptYesNo() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("Expected answer to be '%t', but got '%t'", test.want, got)
			}
		})
	}
}
