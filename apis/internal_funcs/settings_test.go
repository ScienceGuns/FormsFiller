/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package internal_funcs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSettingsFileProblem validates that settingsFileProblem catches missing files, directories and bad modes
func TestSettingsFileProblem(t *testing.T) {
	tempDirectory := t.TempDir()

	// Create a settings file with the correct mode
	goodSettingsPath := filepath.Join(tempDirectory, "good.json")
	if err := os.WriteFile(goodSettingsPath, []byte("{}"), 0400); err != nil {
		t.Fatalf("Failed to write good settings file: %v", err)
	}
	// Create a settings file that other users could read
	openSettingsPath := filepath.Join(tempDirectory, "open.json")
	if err := os.WriteFile(openSettingsPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write open settings file: %v", err)
	}
	// Create a directory where the settings file should be
	directorySettingsPath := filepath.Join(tempDirectory, "directory.json")
	if err := os.Mkdir(directorySettingsPath, 0700); err != nil {
		t.Fatalf("Failed to create settings directory: %v", err)
	}

	tests := []struct {
		name             string
		settingsFilePath string
		wantErr          bool
	}{
		{"Mode 0400", goodSettingsPath, false},
		{"Mode 0644", openSettingsPath, true},
		{"Directory", directorySettingsPath, true},
		{"Missing", filepath.Join(tempDirectory, "missing.json"), true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := settingsFileProblem(test.settingsFilePath)
			if (err != nil) != test.wantErr {
				t.Errorf("settingsFileProblem() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestReadSettingsFile validates that a complete settings file is read and validated
func TestReadSettingsFile(t *testing.T) {
	// Set a fixed point in time (Archer's original air date) for the date of birth check
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()

	// Write Dr. Krieger's settings file
	tempDirectory := t.TempDir()
	settingsFilePath := filepath.Join(tempDirectory, "formsfiller.json")
	settingsJSON := fmt.Sprintf(`{
  "settings": {
    "output-path": "%s"
  },
  "user-data": {
    "trust-name": "The Krieger Clone Trust",
    "trust-address": "1 Figgis Agency Plaza, New York, NY 10001",
    "full-legal-name": "Algernop Krieger",
    "full-legal-address": "1 Figgis Agency Plaza, New York, NY 10001",
    "phone-number": "212-555-0187",
    "email-address": "dr.krieger@figgis.example",
    "other-names-used": "Dr. Krieger",
    "social-security-number": "078-05-1120",
    "date-of-birth": "09/24/1977",
    "ethnicity": "Not Hispanic or Latino",
    "race": ["White"],
    "state-of-birth": "New York",
    "country-of-birth": "United States of America"
  },
  "cleo": {
    "agency-name": "New York City Police Department",
    "official-name-and-title": "Barry Dylan, Chief of Police",
    "mailing-address": "1 Police Plaza, New York, NY 10007"
  }
}`, tempDirectory)
	if err := os.WriteFile(settingsFilePath, []byte(settingsJSON), 0400); err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	settings, err := ReadSettingsFile(settingsFilePath)
	if err != nil {
		t.Fatalf("ReadSettingsFile() returned an unexpected error: %v", err)
	}
	if settings.Settings.OutputPath != tempDirectory {
		t.Errorf("Expected output path to be '%s', but got '%s'", tempDirectory, settings.Settings.OutputPath)
	}
	if settings.UserData.TrustName != "The Krieger Clone Trust" {
		t.Errorf("Expected trust name to be '%s', but got '%s'", "The Krieger Clone Trust", settings.UserData.TrustName)
	}
	if len(settings.UserData.Race) != 1 || settings.UserData.Race[0] != "White" {
		t.Errorf("Expected race to be '%s', but got '%v'", "[White]", settings.UserData.Race)
	}
	if settings.CLEO.OfficialNameAndTitle != "Barry Dylan, Chief of Police" {
		t.Errorf("Expected CLEO official to be '%s', but got '%s'", "Barry Dylan, Chief of Police", settings.CLEO.OfficialNameAndTitle)
	}
}

// TestReadSettingsFile_DefaultOutputPath validates that a missing output-path falls back to the default
func TestReadSettingsFile_DefaultOutputPath(t *testing.T) {
	// Set a fixed point in time (Archer's original air date) for the date of birth check
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()

	// Point the default at a temp directory since a CI runner may not have a Desktop
	tempDirectory := t.TempDir()
	realDefaultOutputPath := defaultOutputPath
	defaultOutputPath = tempDirectory
	defer func() { defaultOutputPath = realDefaultOutputPath }()

	// Write Dr. Krieger's settings file without an output path
	settingsFilePath := filepath.Join(tempDirectory, "formsfiller.json")
	settingsJSON := `{
  "user-data": {
    "full-legal-name": "Algernop Krieger",
    "full-legal-address": "1 Figgis Agency Plaza, New York, NY 10001",
    "phone-number": "212-555-0187",
    "email-address": "dr.krieger@figgis.example",
    "social-security-number": "078-05-1120",
    "date-of-birth": "09/24/1977",
    "ethnicity": "Not Hispanic or Latino",
    "race": ["White"],
    "state-of-birth": "New York",
    "country-of-birth": "United States of America"
  },
  "cleo": {
    "agency-name": "New York City Police Department",
    "official-name-and-title": "Barry Dylan, Chief of Police",
    "mailing-address": "1 Police Plaza, New York, NY 10007"
  }
}`
	if err := os.WriteFile(settingsFilePath, []byte(settingsJSON), 0400); err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	settings, err := ReadSettingsFile(settingsFilePath)
	if err != nil {
		t.Fatalf("ReadSettingsFile() returned an unexpected error: %v", err)
	}
	if settings.Settings.OutputPath != tempDirectory {
		t.Errorf("Expected output path to be '%s', but got '%s'", tempDirectory, settings.Settings.OutputPath)
	}
}

// TestReadSettingsFile_UnknownKey validates that a misspelled key is rejected instead of silently ignored
func TestReadSettingsFile_UnknownKey(t *testing.T) {
	// Write a settings file using snake_case for the phone number
	tempDirectory := t.TempDir()
	settingsFilePath := filepath.Join(tempDirectory, "formsfiller.json")
	settingsJSON := `{
  "user-data": {
    "phone_number": "212-555-0187"
  }
}`
	if err := os.WriteFile(settingsFilePath, []byte(settingsJSON), 0400); err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	_, err := ReadSettingsFile(settingsFilePath)
	if err == nil {
		t.Fatalf("Expected ReadSettingsFile() to return an error, but got nil")
	}
	if !strings.Contains(err.Error(), "phone_number") {
		t.Errorf("Expected error to mention '%s', but got '%s'", "phone_number", err.Error())
	}
}

// TestExpandHomePath validates that only a leading ~ is expanded to the home directory
func TestExpandHomePath(t *testing.T) {
	homeDirectory, err := OutputUserPath("")
	if err != nil {
		t.Fatalf("Failed to look up the home directory: %v", err)
	}

	tests := []struct {
		name         string
		pathToExpand string
		want         string
	}{
		{"Bare tilde", "~", homeDirectory},
		{"Tilde prefix", "~/Desktop", filepath.Join(homeDirectory, "Desktop")},
		{"Absolute path", "/tmp/isis", "/tmp/isis"},
		{"Tilde in the middle", "/tmp/~/isis", "/tmp/~/isis"},
		{"Another user's home", "~cyril/Desktop", "~cyril/Desktop"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := expandHomePath(test.pathToExpand)
			if err != nil {
				t.Fatalf("expandHomePath() returned an unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("Expected path to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestValidateSettingsFile validates the cross-field and required-key rules of validateSettingsFile
func TestValidateSettingsFile(t *testing.T) {
	// Set a fixed point in time (Archer's original air date) for the date of birth check
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()

	tempDirectory := t.TempDir()
	// Create a file to use where the output directory should be
	notADirectoryPath := filepath.Join(tempDirectory, "kinda-a-directory")
	if err := os.WriteFile(notADirectoryPath, []byte("Danger Zone"), 0600); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// validSettings Build a fresh copy of Dr. Krieger's valid settings for each test to break
	validSettings := func() FFConfig {
		var settings FFConfig
		settings.Settings.OutputPath = tempDirectory
		settings.UserData.TrustName = "The Krieger Clone Trust"
		settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"
		settings.UserData.FullLegalName = "Algernop Krieger"
		settings.UserData.FullLegalAddress = "1 Figgis Agency Plaza, New York, NY 10001"
		settings.UserData.PhoneNumber = "212-555-0187"
		settings.UserData.EmailAddress = "dr.krieger@figgis.example"
		settings.UserData.SocialSecurityNumber = "078-05-1120"
		settings.UserData.DateOfBirth = "09/24/1977"
		settings.UserData.Ethnicity = "Not Hispanic or Latino"
		settings.UserData.Race = []string{"White"}
		settings.UserData.StateOfBirth = "New York"
		settings.UserData.CountryOfBirth = "United States of America"
		settings.CLEO.AgencyName = "New York City Police Department"
		settings.CLEO.OfficialNameAndTitle = "Barry Dylan, Chief of Police"
		settings.CLEO.MailingAddress = "1 Police Plaza, New York, NY 10007"
		return settings
	}

	tests := []struct {
		name    string
		breakIt func(settings *FFConfig)
		wantErr bool
	}{
		{"Valid", func(settings *FFConfig) {}, false},
		{"No trust at all", func(settings *FFConfig) {
			settings.UserData.TrustName = ""
			settings.UserData.TrustAddress = ""
		}, false},
		{"Trust name without an address", func(settings *FFConfig) { settings.UserData.TrustAddress = "" }, true},
		{"Trust address without a name", func(settings *FFConfig) { settings.UserData.TrustName = "" }, true},
		{"Missing output path", func(settings *FFConfig) { settings.Settings.OutputPath = filepath.Join(tempDirectory, "nope") }, true},
		{"Output path is a file", func(settings *FFConfig) { settings.Settings.OutputPath = notADirectoryPath }, true},
		{"Missing full legal name", func(settings *FFConfig) { settings.UserData.FullLegalName = "" }, true},
		{"Whitespace state of birth", func(settings *FFConfig) { settings.UserData.StateOfBirth = "   " }, true},
		{"Missing CLEO address", func(settings *FFConfig) { settings.CLEO.MailingAddress = "" }, true},
		{"Bad phone number", func(settings *FFConfig) { settings.UserData.PhoneNumber = "2125550187" }, true},
		{"No race", func(settings *FFConfig) { settings.UserData.Race = nil }, true},
		{"Other country of citizenship", func(settings *FFConfig) { settings.UserData.CountryOfCitizenship = "Brazil" }, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := validSettings()
			test.breakIt(&settings)
			err := validateSettingsFile(settings)
			if (err != nil) != test.wantErr {
				t.Errorf("validateSettingsFile() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidatePhoneNumber validates the 123-456-7890 phone number format
func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name        string
		phoneNumber string
		wantErr     bool
	}{
		{"Valid", "212-555-0187", false},
		{"No dashes", "2125550187", true},
		{"Parentheses", "(212) 555-0187", true},
		{"Dots", "212.555.0187", true},
		{"Country code", "+1-212-555-0187", true},
		{"Letters", "212-555-LANA", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePhoneNumber(test.phoneNumber)
			if (err != nil) != test.wantErr {
				t.Errorf("validatePhoneNumber() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateEmailAddress validates that only bare e-mail addresses are accepted
func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name         string
		emailAddress string
		wantErr      bool
	}{
		{"Valid", "dr.krieger@figgis.example", false},
		{"Display name", "Dr. Krieger <dr.krieger@figgis.example>", true},
		{"No at sign", "dr.krieger.figgis.example", true},
		{"No domain", "dr.krieger@", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateEmailAddress(test.emailAddress)
			if (err != nil) != test.wantErr {
				t.Errorf("validateEmailAddress() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateSocialSecurityNumber validates the SSN format and the number ranges the SSA never issues
func TestValidateSocialSecurityNumber(t *testing.T) {
	tests := []struct {
		name                 string
		socialSecurityNumber string
		wantErr              bool
	}{
		{"Valid", "078-05-1120", false},
		{"No dashes", "078051120", true},
		{"Too short", "078-05-112", true},
		{"Area 000", "000-05-1120", true},
		{"Area 666", "666-05-1120", true},
		{"Area 9xx", "912-05-1120", true},
		{"Group 00", "078-00-1120", true},
		{"Serial 0000", "078-05-0000", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSocialSecurityNumber(test.socialSecurityNumber)
			if (err != nil) != test.wantErr {
				t.Errorf("validateSocialSecurityNumber() error = %v, wantErr %v", err, test.wantErr)
			}
			// The SSN must never show up in an error message
			if err != nil && strings.Contains(err.Error(), test.socialSecurityNumber) {
				t.Errorf("Expected error to not contain the SSN, but got '%s'", err.Error())
			}
		})
	}
}

// TestValidateDateOfBirth validates the MM/DD/YYYY format, real dates and dates in the past
func TestValidateDateOfBirth(t *testing.T) {
	// Set a fixed point in time (Archer's original air date) so "the future" doesn't move
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()

	tests := []struct {
		name        string
		dateOfBirth string
		wantErr     bool
	}{
		{"Valid", "09/24/1977", false},
		{"Leap day", "02/29/1972", false},
		{"No leading zeroes", "9/4/1977", true},
		{"Two digit year", "09/24/77", true},
		{"ISO format", "1977-09-24", true},
		{"Not a real date", "02/30/1977", true},
		{"Not a leap year", "02/29/1971", true},
		{"Month 13", "13/24/1977", true},
		{"Archer air date", "09/17/2009", false},
		{"Day after the air date", "09/18/2009", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDateOfBirth(test.dateOfBirth)
			if (err != nil) != test.wantErr {
				t.Errorf("validateDateOfBirth() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateEthnicity validates that only the ATF ethnicity values are accepted
func TestValidateEthnicity(t *testing.T) {
	tests := []struct {
		name      string
		ethnicity string
		wantErr   bool
	}{
		{"Hispanic or Latino", "Hispanic or Latino", false},
		{"Not Hispanic or Latino", "Not Hispanic or Latino", false},
		{"Wrong case", "not hispanic or latino", true},
		{"Made up", "Clone", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateEthnicity(test.ethnicity)
			if (err != nil) != test.wantErr {
				t.Errorf("validateEthnicity() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateRace validates that race lists are non-empty, use ATF values and have no duplicates
func TestValidateRace(t *testing.T) {
	tests := []struct {
		name    string
		races   []string
		wantErr bool
	}{
		{"One race", []string{"White"}, false},
		{"Two races", []string{"Asian", "White"}, false},
		{"Empty", []string{}, true},
		{"Made up", []string{"Cyborg"}, true},
		{"Duplicate", []string{"White", "White"}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRace(test.races)
			if (err != nil) != test.wantErr {
				t.Errorf("validateRace() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateCountryOfCitizenship validates that U.S. values are rejected since blank means U.S.A.
func TestValidateCountryOfCitizenship(t *testing.T) {
	tests := []struct {
		name                 string
		countryOfCitizenship string
		wantErr              bool
	}{
		{"Blank", "", false},
		{"Other country", "Brazil", false},
		{"USA", "USA", true},
		{"Spelled out", "United States of America", true},
		{"Padded lowercase", "  united states ", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCountryOfCitizenship(test.countryOfCitizenship)
			if (err != nil) != test.wantErr {
				t.Errorf("validateCountryOfCitizenship() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateStateOfBirth validates that state of birth is only required for people born in the U.S.
func TestValidateStateOfBirth(t *testing.T) {
	tests := []struct {
		name           string
		stateOfBirth   string
		countryOfBirth string
		wantErr        bool
	}{
		{"U.S. with a state", "New York", "United States of America", false},
		{"U.S. without a state", "", "United States of America", true},
		{"USA abbreviation without a state", "", "USA", true},
		{"U.S. with a blank state", "   ", "united states", true},
		{"Born abroad without a state", "", "Brazil", false},
		{"Born abroad with a province", "Rio de Janeiro", "Brazil", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateStateOfBirth(test.stateOfBirth, test.countryOfBirth)
			if (err != nil) != test.wantErr {
				t.Errorf("validateStateOfBirth() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
