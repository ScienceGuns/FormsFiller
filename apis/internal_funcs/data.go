/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package internal_funcs

import (
	"regexp"
	"time"
)

// FFConfig A struct to keep track of known values in formsfiller.json
type FFConfig struct {
	Settings struct {
		OutputPath string `json:"output-path,omitempty"`
	} `json:"settings"`
	UserData struct {
		TrustName            string   `json:"trust-name,omitempty"`
		TrustAddress         string   `json:"trust-address,omitempty"`
		FullLegalName        string   `json:"full-legal-name"`
		FullLegalAddress     string   `json:"full-legal-address"`
		PhoneNumber          string   `json:"phone-number"`
		EmailAddress         string   `json:"email-address"`
		OtherNamesUsed       string   `json:"other-names-used,omitempty"`
		SocialSecurityNumber string   `json:"social-security-number"`
		DateOfBirth          string   `json:"date-of-birth"`
		Ethnicity            string   `json:"ethnicity"`
		Race                 []string `json:"race"`
		CountryOfCitizenship string   `json:"country-of-citizenship,omitempty"`
		StateOfBirth         string   `json:"state-of-birth,omitempty"`
		CountryOfBirth       string   `json:"country-of-birth"`
	} `json:"user-data"`
	CLEO struct {
		AgencyName           string `json:"agency-name"`
		OfficialNameAndTitle string `json:"official-name-and-title"`
		MailingAddress       string `json:"mailing-address"`
	} `json:"cleo"`
}

// now is a variable that can be overridden for testing so date of birth checks are predictable
var now = time.Now

// defaultOutputPath The output path used when settings.output-path is not set
var defaultOutputPath, _ = OutputUserPath("Desktop")

// validEthnicities The ethnicity values the ATF forms accept
var validEthnicities = []string{
	"Hispanic or Latino",
	"Not Hispanic or Latino",
}

// validRaces The race values the ATF forms accept
var validRaces = []string{
	"American Indian or Alaska Native",
	"Asian",
	"Black or African American",
	"Native Hawaiian or Other Pacific Islander",
	"White",
}

// unitedStatesNames Country values that mean the U.S.A., used for country of birth and country of citizenship
var unitedStatesNames = []string{
	"united states",
	"united states of america",
	"usa",
	"us",
	"u.s.",
	"u.s.a.",
}

// phoneNumberPattern The phone number format the ATF forms expect (123-456-7890)
var phoneNumberPattern = regexp.MustCompile(`^\d{3}-\d{3}-\d{4}$`)

// socialSecurityNumberPattern The U.S. SSN format split into its area, group and serial parts
var socialSecurityNumberPattern = regexp.MustCompile(`^(\d{3})-(\d{2})-(\d{4})$`)

// dateOfBirthPattern The MM/DD/YYYY format with leading zeroes for single digit months and days
var dateOfBirthPattern = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)

// dateOfBirthLayout The Go time layout matching dateOfBirthPattern
var dateOfBirthLayout = "01/02/2006"
