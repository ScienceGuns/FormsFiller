/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bufio"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestCheckRPQSettings validates that Form 5320.23 is only available when a trust name is set
func TestCheckRPQSettings(t *testing.T) {
	tests := []struct {
		name      string
		trustName string
		wantErr   bool
	}{
		{"With a trust", "The Krieger Clone Trust", false},
		{"Without a trust", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var settings internal_funcs.FFConfig
			settings.UserData.TrustName = test.trustName
			err := checkRPQSettings(&settings)
			if (err != nil) != test.wantErr {
				t.Errorf("checkRPQSettings() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestPromptRPQAnswers validates that each prompt's answer lands in the right place, including after bad input
func TestPromptRPQAnswers(t *testing.T) {
	// Form 4, a bad firearm type then Short-Barreled Rifle, a blank maker name, then The Gauss SBR
	input := strings.Join([]string{
		"2",
		"7",
		"1",
		"",
		"Mad Scientist Research LLC",
		"Perkiomenville, PA 18074",
		"The Gauss",
		"300",
		"TGB126MS0001",
	}, "\n") + "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	answers, err := promptRPQAnswers(reader)
	if err != nil {
		t.Fatalf("promptRPQAnswers() returned an unexpected error: %v", err)
	}
	checks := []struct {
		field string
		want  string
		got   string
	}{
		{"ATF form field", "form4[0]", answers.ATFForm.FieldName},
		{"firearm type", "Short-Barreled Rifle", answers.FirearmType},
		{"maker name", "Mad Scientist Research LLC", answers.MakerName},
		{"maker address", "Perkiomenville, PA 18074", answers.MakerAddress},
		{"model", "The Gauss", answers.Model},
		{"caliber", "300", answers.Caliber},
		{"serial number", "TGB126MS0001", answers.SerialNumber},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("Expected %s to be '%s', but got '%s'", check.field, check.want, check.got)
		}
	}
}

// TestPromptRPQAnswers_InputEnds validates that running out of input stops the prompts with an error
func TestPromptRPQAnswers_InputEnds(t *testing.T) {
	// Input stops after the model
	input := "2\n1\nMad Scientist Research LLC\nPerkiomenville, PA 18074\nThe Gauss\n"
	reader := bufio.NewReader(strings.NewReader(input))
	if _, err := promptRPQAnswers(reader); err == nil {
		t.Errorf("Expected promptRPQAnswers() to return an error, but got nil")
	}
}

// TestFillRPQ validates that answering no at the review asks every question again, then both copies are saved
func TestFillRPQ(t *testing.T) {
	// Keep pdfcpu's config and the finished forms in temp directories
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	var settings internal_funcs.FFConfig
	settings.Settings.OutputPath = t.TempDir()
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}

	// The first pass is rejected at the review and the second is accepted with Enter
	onePass := "2\n1\nMad Scientist Research LLC\nPerkiomenville, PA 18074\nThe Gauss\n300\nTGB126MS0001\n"
	input := onePass + "n\n" + onePass + "\n"
	reader := bufio.NewReader(strings.NewReader(input))
	if err := fillRPQ(reader, &settings); err != nil {
		t.Fatalf("fillRPQ() returned an unexpected error: %v", err)
	}
	// Every line of input should have been used
	if leftover, _ := reader.ReadString('\n'); leftover != "" {
		t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
	}

	// Both copies should be saved, readable only by the owner
	for _, copyName := range []string{"ATF", "CLEO"} {
		outputPath := filepath.Join(settings.Settings.OutputPath, "TGB126MS0001_TheKriegerCloneTrust_Krieger_RPQ_"+copyName+".pdf")
		outputFile, err := os.Stat(outputPath)
		if err != nil {
			t.Errorf("Expected %s to exist, but got: %v", outputPath, err)
			continue
		}
		if outputFile.Mode().Perm() != 0600 {
			t.Errorf("Expected %s to have mode '%#o', but got '%#o'", outputPath, 0600, outputFile.Mode().Perm())
		}
	}
}

// TestFillRPQ_KeepExisting validates that declining to overwrite an existing copy leaves it alone and saves nothing
func TestFillRPQ_KeepExisting(t *testing.T) {
	// Keep pdfcpu's config and the finished forms in temp directories
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	var settings internal_funcs.FFConfig
	settings.Settings.OutputPath = t.TempDir()
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}

	// An ATF copy from an earlier run is already there
	atfPath := filepath.Join(settings.Settings.OutputPath, "TGB126MS0001_TheKriegerCloneTrust_Krieger_RPQ_ATF.pdf")
	if err := os.WriteFile(atfPath, []byte("Danger Zone"), 0600); err != nil {
		t.Fatalf("Failed to write existing ATF copy: %v", err)
	}

	// Confirm the answers, then press Enter to take the default of not overwriting
	input := "2\n1\nMad Scientist Research LLC\nPerkiomenville, PA 18074\nThe Gauss\n300\nTGB126MS0001\n\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	if err := fillRPQ(reader, &settings); err != nil {
		t.Fatalf("fillRPQ() returned an unexpected error: %v", err)
	}

	// The existing ATF copy is untouched and no CLEO copy was written
	atfData, err := os.ReadFile(atfPath)
	if err != nil {
		t.Fatalf("Failed to read the existing ATF copy: %v", err)
	}
	if string(atfData) != "Danger Zone" {
		t.Errorf("Expected the ATF copy to be '%s', but got '%s'", "Danger Zone", string(atfData))
	}
	cleoPath := filepath.Join(settings.Settings.OutputPath, "TGB126MS0001_TheKriegerCloneTrust_Krieger_RPQ_CLEO.pdf")
	if _, err := os.Stat(cleoPath); err == nil {
		t.Errorf("Expected %s to not exist, but it does", cleoPath)
	}
}

// TestRPQFilePath validates the {Serial}_{Trust}_{LastName}_RPQ_{Copy}.pdf file name with spaces stripped
func TestRPQFilePath(t *testing.T) {
	var settings internal_funcs.FFConfig
	settings.Settings.OutputPath = "/tmp/isis"
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.FullLegalName = "Algernop Krieger"
	answers := rpqAnswers{SerialNumber: "TGB126MS0001"}

	want := "/tmp/isis/TGB126MS0001_TheKriegerCloneTrust_Krieger_RPQ_CLEO.pdf"
	if got := rpqFilePath(&settings, answers, "CLEO"); got != want {
		t.Errorf("Expected file path to be '%s', but got '%s'", want, got)
	}
}

// TestBuildRPQFields validates the field values on the ATF copy of Dr. Krieger's 5320.23
func TestBuildRPQFields(t *testing.T) {
	// Dr. Krieger's settings
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.FullLegalAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.UserData.PhoneNumber = "212-555-0187"
	settings.UserData.EmailAddress = "dr.krieger@figgis.example"
	settings.UserData.OtherNamesUsed = "Dr. Krieger"
	settings.UserData.SocialSecurityNumber = "078-05-1120"
	settings.UserData.DateOfBirth = "09/24/1977"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"Asian", "White"}
	settings.UserData.StateOfBirth = "New York"
	settings.UserData.CountryOfBirth = "United States of America"
	settings.CLEO.AgencyName = "New York City Police Department"
	settings.CLEO.OfficialNameAndTitle = "Barry Dylan, Chief of Police"
	settings.CLEO.MailingAddress = "1 Police Plaza, New York, NY 10007"
	// The Gauss SBR from Mad Scientist Research
	answers := rpqAnswers{
		ATFForm:      rpqATFFormOptions[1],
		FirearmType:  "Short-Barreled Rifle",
		MakerName:    "Mad Scientist Research LLC",
		MakerAddress: "Perkiomenville, PA 18074",
		Model:        "The Gauss",
		Caliber:      "300",
		SerialNumber: "TGB126MS0001",
	}

	fields := buildRPQFields(&settings, answers)

	// Check the text fields on both ATF pages
	page1 := "topmostSubform[0].Page1[0]."
	page2 := "topmostSubform[0].Page2[0]."
	wantText := map[string]string{
		page1 + "applicantaddress[0]":   "The Krieger Clone Trust\n1 Figgis Agency Plaza, New York, NY 10001",
		page1 + "responsibleaddress[0]": "Algernop Krieger\n1 Figgis Agency Plaza, New York, NY 10001",
		page1 + "telephone[0]":          "212-555-0187",
		page1 + "email[0]":              "dr.krieger@figgis.example",
		page1 + "othernames[0]":         "Dr. Krieger",
		page1 + "ssn2f[0]":              "078-05-1120",
		page1 + "#field[24]":            "09/24/1977",
		page1 + "firearmtype[0]":        "Short-Barreled Rifle",
		page1 + "importeraddress[0]":    "Mad Scientist Research LLC\nPerkiomenville, PA 18074",
		page1 + "Model[0]":              "The Gauss",
		page1 + "caliber[0]":            "300",
		page1 + "serial[0]":             "TGB126MS0001",
		page1 + "TextField3[0]":         "New York City Police Department",
		page1 + "TextField4[0]":         "Barry Dylan, Chief of Police",
		page1 + "TextField5[0]":         "1 Police Plaza, New York, NY 10007",
		page2 + "statebirth[0]":         "New York",
		page2 + "statecountry[0]":       "United States of America",
	}
	for fieldName, want := range wantText {
		if got := fields.TextFields[fieldName]; got != want {
			t.Errorf("Expected %s to be '%s', but got '%s'", fieldName, want, got)
		}
	}

	// Check the boxes that should be checked on both ATF pages
	wantChecked := []string{
		page1 + "form4[0]",
		page1 + "nhl[0]",
		page1 + "a[0]",
		page1 + "w[0]",
		page2 + "CheckBoxno6a[0]",
		page2 + "CheckBoxno11[0]",
		page2 + "CheckBoxNA[0]",
		page2 + "no17[0]",
		page2 + "usacheckbox[0]",
	}
	for _, fieldName := range wantChecked {
		if !slices.Contains(fields.CheckedBoxes, fieldName) {
			t.Errorf("Expected %s to be checked, but it wasn't", fieldName)
		}
	}

	// No Yes box should ever be checked, and other country shouldn't be checked for a U.S. citizen
	for _, fieldName := range fields.CheckedBoxes {
		if strings.Contains(fieldName, "Yes") || strings.Contains(fieldName, "yes17") || strings.Contains(fieldName, "othercountry") {
			t.Errorf("Expected %s to not be checked, but it was", fieldName)
		}
	}
}

// TestPromptRPQAnswers_BlankMakerAddress validates that the maker address can be left blank after the warning
func TestPromptRPQAnswers_BlankMakerAddress(t *testing.T) {
	// Leave the maker address blank twice
	input := "2\n1\nMad Scientist Research LLC\n\n\nThe Gauss\n300\nTGB126MS0001\n"
	reader := bufio.NewReader(strings.NewReader(input))

	answers, err := promptRPQAnswers(reader)
	if err != nil {
		t.Fatalf("promptRPQAnswers() returned an unexpected error: %v", err)
	}
	if answers.MakerAddress != "" {
		t.Errorf("Expected maker address to be '%s', but got '%s'", "", answers.MakerAddress)
	}
	if answers.Model != "The Gauss" {
		t.Errorf("Expected model to be '%s', but got '%s'", "The Gauss", answers.Model)
	}
}

// TestBuildRPQFields_BlankMakerAddress validates that Item 4b is just the maker's name when there's no address
func TestBuildRPQFields_BlankMakerAddress(t *testing.T) {
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}
	answers := rpqAnswers{ATFForm: rpqATFFormOptions[1], MakerName: "Mad Scientist Research LLC"}

	fields := buildRPQFields(&settings, answers)

	fieldName := "topmostSubform[0].Page1[0].importeraddress[0]"
	if got := fields.TextFields[fieldName]; got != "Mad Scientist Research LLC" {
		t.Errorf("Expected %s to be '%s', but got '%s'", fieldName, "Mad Scientist Research LLC", got)
	}
}

// TestBuildRPQFields_CLEOCopy validates that the CLEO copy is filled out but never gets the SSN or serial number
func TestBuildRPQFields_CLEOCopy(t *testing.T) {
	// Dr. Krieger's settings
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.SocialSecurityNumber = "078-05-1120"
	settings.UserData.DateOfBirth = "09/24/1977"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}
	answers := rpqAnswers{ATFForm: rpqATFFormOptions[1], SerialNumber: "TGB126MS0001"}

	fields := buildRPQFields(&settings, answers)

	// The CLEO copy uses its own date of birth field and still gets the Item 1 and Item 6 boxes
	page5 := "topmostSubform[0].Page5[0]."
	page6 := "topmostSubform[0].Page6[0]."
	if got := fields.TextFields[page5+"#field[22]"]; got != "09/24/1977" {
		t.Errorf("Expected CLEO date of birth to be '%s', but got '%s'", "09/24/1977", got)
	}
	for _, fieldName := range []string{page5 + "form4[0]", page6 + "CheckBoxNA[0]"} {
		if !slices.Contains(fields.CheckedBoxes, fieldName) {
			t.Errorf("Expected %s to be checked, but it wasn't", fieldName)
		}
	}

	// The SSN and serial number must not be anywhere on pages 5 or 6
	for fieldName, value := range fields.TextFields {
		onCLEOCopy := strings.HasPrefix(fieldName, page5) || strings.HasPrefix(fieldName, page6)
		if onCLEOCopy && (value == "078-05-1120" || value == "TGB126MS0001") {
			t.Errorf("Expected the CLEO copy to have no SSN or serial number, but %s is '%s'", fieldName, value)
		}
	}
}

// TestBuildRPQFields_OtherCitizenship validates that a country of citizenship replaces the U.S.A. box
func TestBuildRPQFields_OtherCitizenship(t *testing.T) {
	// Dr. Krieger's settings as a citizen of Brazil
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}
	settings.UserData.CountryOfCitizenship = "Brazil"
	answers := rpqAnswers{ATFForm: rpqATFFormOptions[1]}

	fields := buildRPQFields(&settings, answers)

	page2 := "topmostSubform[0].Page2[0]."
	if !slices.Contains(fields.CheckedBoxes, page2+"othercountrycheckbox[0]") {
		t.Errorf("Expected %s to be checked, but it wasn't", page2+"othercountrycheckbox[0]")
	}
	if slices.Contains(fields.CheckedBoxes, page2+"usacheckbox[0]") {
		t.Errorf("Expected %s to not be checked, but it was", page2+"usacheckbox[0]")
	}
	if got := fields.TextFields[page2+"Othercountry[0]"]; got != "Brazil" {
		t.Errorf("Expected other country to be '%s', but got '%s'", "Brazil", got)
	}
}
