/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package internal_funcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// CheckForSettingsFile Check that the settings file exists and has the correct mode, exiting if it doesn't
func CheckForSettingsFile(settingsFilePath string) {
	if err := settingsFileProblem(settingsFilePath); err != nil {
		log.Fatalf("Settings file problem: %v", err)
	}
}

// settingsFileProblem Return the first problem with the settings file's existence, type or mode
func settingsFileProblem(settingsFilePath string) error {
	// Make sure the file is there at all
	settingsFile, err := os.Stat(settingsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist. See the README for an example settings file", settingsFilePath)
		}
		return fmt.Errorf("failed to check for %s: %w", settingsFilePath, err)
	}
	// A directory or other special file is not a settings file
	if !settingsFile.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", settingsFilePath)
	}
	// The settings file holds PII like an SSN and date of birth so only the owner may read it
	if settingsFile.Mode().Perm() != 0400 {
		return fmt.Errorf("%s has mode %#o but must be 0400. Run `chmod 0400 %s` and treat its contents as exposed to anyone else on this machine",
			settingsFilePath, settingsFile.Mode().Perm(), settingsFilePath)
	}
	return nil
}

// ReadSettingsFile Read the settings file, apply defaults and validate the result
func ReadSettingsFile(settingsFilePath string) (*FFConfig, error) {
	// Open the settings file
	settingsFile, err := os.Open(settingsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the settings file: %w", err)
	}
	defer func() {
		if err := settingsFile.Close(); err != nil {
			log.Printf("Warning: Failed to close settingsFile: %v", err)
		}
	}()

	// Decode the JSON, rejecting unknown keys so a typo doesn't silently leave a field blank
	var outputConfig FFConfig
	jsonDecoder := json.NewDecoder(settingsFile)
	jsonDecoder.DisallowUnknownFields()
	if err := jsonDecoder.Decode(&outputConfig); err != nil {
		return nil, fmt.Errorf("failed to parse the settings file: %w", err)
	}

	// Default the output path to the user's Desktop
	if outputConfig.Settings.OutputPath == "" {
		outputConfig.Settings.OutputPath = defaultOutputPath
	}
	// Expand a leading ~ so the output path can be written the same way it would be in a shell
	expandedPath, err := expandHomePath(outputConfig.Settings.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand the output path: %w", err)
	}
	outputConfig.Settings.OutputPath = expandedPath

	// Make sure everything in the file is sane before anything uses it
	if err := validateSettingsFile(outputConfig); err != nil {
		return nil, fmt.Errorf("failed to validate the settings file: %w", err)
	}
	return &outputConfig, nil
}

// OutputUserPath Output a path under the current user's home directory
func OutputUserPath(userPath string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to determine the current user: %w", err)
	}
	return filepath.Join(currentUser.HomeDir, userPath), nil
}

// expandHomePath Replace a leading ~ in a path with the current user's home directory
func expandHomePath(pathToExpand string) (string, error) {
	// Only a bare ~ or a ~/ prefix means the current user's home directory
	if pathToExpand != "~" && !strings.HasPrefix(pathToExpand, "~/") {
		return pathToExpand, nil
	}
	return OutputUserPath(strings.TrimPrefix(pathToExpand, "~"))
}

// validateSettingsFile Validate that the contents of the settings file are sane
func validateSettingsFile(settings FFConfig) error {
	// The output path has to be an existing directory
	outputDirectory, err := os.Stat(settings.Settings.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to find output-path %s: %w", settings.Settings.OutputPath, err)
	}
	if !outputDirectory.IsDir() {
		return fmt.Errorf("output-path %s is not a directory", settings.Settings.OutputPath)
	}

	// A trust name is optional, but a trust needs an address and an address needs a trust
	if settings.UserData.TrustName != "" && settings.UserData.TrustAddress == "" {
		return errors.New("trust-address is required when trust-name is set")
	}
	if settings.UserData.TrustName == "" && settings.UserData.TrustAddress != "" {
		return errors.New("trust-address is set but trust-name is blank")
	}

	// Check every required key has a value
	requiredKeys := []struct {
		key   string
		value string
	}{
		{"user-data.full-legal-name", settings.UserData.FullLegalName},
		{"user-data.full-legal-address", settings.UserData.FullLegalAddress},
		{"user-data.phone-number", settings.UserData.PhoneNumber},
		{"user-data.email-address", settings.UserData.EmailAddress},
		{"user-data.social-security-number", settings.UserData.SocialSecurityNumber},
		{"user-data.date-of-birth", settings.UserData.DateOfBirth},
		{"user-data.ethnicity", settings.UserData.Ethnicity},
		{"user-data.country-of-birth", settings.UserData.CountryOfBirth},
		{"cleo.agency-name", settings.CLEO.AgencyName},
		{"cleo.official-name-and-title", settings.CLEO.OfficialNameAndTitle},
		{"cleo.mailing-address", settings.CLEO.MailingAddress},
	}
	for _, requiredKey := range requiredKeys {
		if strings.TrimSpace(requiredKey.value) == "" {
			return fmt.Errorf("%s is required", requiredKey.key)
		}
	}

	// Check the values that have a specific format
	if err := validatePhoneNumber(settings.UserData.PhoneNumber); err != nil {
		return err
	}
	if err := validateEmailAddress(settings.UserData.EmailAddress); err != nil {
		return err
	}
	if err := validateSocialSecurityNumber(settings.UserData.SocialSecurityNumber); err != nil {
		return err
	}
	if err := validateDateOfBirth(settings.UserData.DateOfBirth); err != nil {
		return err
	}
	if err := validateEthnicity(settings.UserData.Ethnicity); err != nil {
		return err
	}
	if err := validateRace(settings.UserData.Race); err != nil {
		return err
	}
	if err := validateStateOfBirth(settings.UserData.StateOfBirth, settings.UserData.CountryOfBirth); err != nil {
		return err
	}
	return validateCountryOfCitizenship(settings.UserData.CountryOfCitizenship)
}

// validateStateOfBirth Validate that anyone born in the U.S. lists their state of birth, which is optional for everyone else
func validateStateOfBirth(stateOfBirth string, countryOfBirth string) error {
	bornInUnitedStates := slices.Contains(unitedStatesNames, strings.ToLower(strings.TrimSpace(countryOfBirth)))
	if bornInUnitedStates && strings.TrimSpace(stateOfBirth) == "" {
		return fmt.Errorf("state-of-birth is required when country-of-birth is %q", countryOfBirth)
	}
	return nil
}

// validatePhoneNumber Validate that a phone number uses the 123-456-7890 format
func validatePhoneNumber(phoneNumber string) error {
	if !phoneNumberPattern.MatchString(phoneNumber) {
		return fmt.Errorf("phone-number %q must use the format 123-456-7890", phoneNumber)
	}
	return nil
}

// validateEmailAddress Validate that an e-mail address is a bare address with no display name
func validateEmailAddress(emailAddress string) error {
	parsedAddress, err := mail.ParseAddress(emailAddress)
	if err != nil || parsedAddress.Address != emailAddress {
		return fmt.Errorf("email-address %q is not a valid e-mail address", emailAddress)
	}
	return nil
}

// validateSocialSecurityNumber Validate that an SSN uses the 123-45-6789 format and could have been issued by the SSA
func validateSocialSecurityNumber(socialSecurityNumber string) error {
	// The SSN itself is never put in an error message since errors end up on screen
	ssnParts := socialSecurityNumberPattern.FindStringSubmatch(socialSecurityNumber)
	if ssnParts == nil {
		return errors.New("social-security-number must use the format 123-45-6789")
	}
	// The SSA never issues area 000, 666 or 900-999, group 00 or serial 0000
	area, group, serial := ssnParts[1], ssnParts[2], ssnParts[3]
	if area == "000" || area == "666" || area[0] == '9' {
		return errors.New("social-security-number has an area number (first 3 digits) the SSA never issues")
	}
	if group == "00" {
		return errors.New("social-security-number has a group number (middle 2 digits) of 00, which the SSA never issues")
	}
	if serial == "0000" {
		return errors.New("social-security-number has a serial number (last 4 digits) of 0000, which the SSA never issues")
	}
	return nil
}

// validateDateOfBirth Validate that a date of birth is a real date in the past using MM/DD/YYYY
func validateDateOfBirth(dateOfBirth string) error {
	// Require the leading zeroes before parsing
	if !dateOfBirthPattern.MatchString(dateOfBirth) {
		return fmt.Errorf("date-of-birth %q must use the format MM/DD/YYYY with leading zeroes", dateOfBirth)
	}
	// Parsing catches impossible dates like 02/30
	parsedDate, err := time.Parse(dateOfBirthLayout, dateOfBirth)
	if err != nil {
		return fmt.Errorf("date-of-birth %q is not a real date", dateOfBirth)
	}
	if parsedDate.After(now()) {
		return fmt.Errorf("date-of-birth %q is in the future", dateOfBirth)
	}
	return nil
}

// validateEthnicity Validate that an ethnicity is one of the values the ATF forms accept
func validateEthnicity(ethnicity string) error {
	if !slices.Contains(validEthnicities, ethnicity) {
		return fmt.Errorf("ethnicity %q must be one of: %s", ethnicity, strings.Join(validEthnicities, ", "))
	}
	return nil
}

// validateRace Validate that at least one race is listed and every listed race is one the ATF forms accept
func validateRace(races []string) error {
	if len(races) == 0 {
		return errors.New("race must list at least one value")
	}
	for index, race := range races {
		if !slices.Contains(validRaces, race) {
			return fmt.Errorf("race %q must be one of: %s", race, strings.Join(validRaces, ", "))
		}
		// A repeated race is almost certainly a copy and paste mistake
		if slices.Contains(races[:index], race) {
			return fmt.Errorf("race %q is listed more than once", race)
		}
	}
	return nil
}

// validateCountryOfCitizenship Validate that an optional country of citizenship isn't the U.S.A., which is what a blank value means
func validateCountryOfCitizenship(countryOfCitizenship string) error {
	if slices.Contains(unitedStatesNames, strings.ToLower(strings.TrimSpace(countryOfCitizenship))) {
		return fmt.Errorf("country-of-citizenship %q should be left blank for U.S. citizens", countryOfCitizenship)
	}
	return nil
}
