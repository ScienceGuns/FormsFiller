/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bytes"
	"context"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestClearMultilineFlag validates that the date of birth fields can be made single line and a missing field is an error
func TestClearMultilineFlag(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	tests := []struct {
		name       string
		fieldNames []string
		wantErr    bool
	}{
		{"Both date of birth fields", []string{"topmostSubform[0].Page1[0].#field[24]", "topmostSubform[0].Page5[0].#field[22]"}, false},
		{"A field the template doesn't have", []string{"topmostSubform[0].Page1[0].#field[24]", "topmostSubform[0].Page1[0].#field[934]"}, true},
		{"Just the short name", []string{"#field[24]"}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updatedPDF, err := clearMultilineFlag(rpqTemplate, test.fieldNames)
			if (err != nil) != test.wantErr {
				t.Fatalf("clearMultilineFlag() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && !bytes.HasPrefix(updatedPDF, []byte("%PDF")) {
				t.Errorf("Expected the updated template to be a PDF, but it starts with '%s'", string(updatedPDF[:8]))
			}
		})
	}
}

// TestFillPDF validates that every 5320.23 field for Dr. Krieger's SBR makes it into the PDF, accents included
func TestFillPDF(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	// Dr. Krieger's settings, with an accent in his other names to make sure non-ASCII text can be filled
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.FullLegalAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.UserData.PhoneNumber = "212-555-0187"
	settings.UserData.EmailAddress = "dr.krieger@figgis.example"
	settings.UserData.OtherNamesUsed = "Dr. Krieger, Señor Krieger"
	settings.UserData.SocialSecurityNumber = "078-05-1120"
	settings.UserData.DateOfBirth = "09/24/1977"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}
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

	// fillPDF reads every value back itself, so no error means everything landed
	filledPDF, err := fillPDF(rpqTemplate, buildRPQFields(&settings, answers))
	if err != nil {
		t.Fatalf("fillPDF() returned an unexpected error: %v", err)
	}
	if !bytes.HasPrefix(filledPDF, []byte("%PDF")) {
		t.Errorf("Expected the filled form to be a PDF, but it starts with '%s'", string(filledPDF[:8]))
	}
}

// TestFillPDF_UnknownFields validates that a field the template doesn't have is an error instead of being skipped
func TestFillPDF_UnknownFields(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	tests := []struct {
		name   string
		fields formFields
	}{
		{"Text field", formFields{TextFields: map[string]string{"topmostSubform[0].Page1[0].phrasing[0]": "Phrasing!"}}},
		{"Checkbox", formFields{TextFields: map[string]string{}, CheckedBoxes: []string{"topmostSubform[0].Page1[0].dangerzone[0]"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := fillPDF(rpqTemplate, test.fields); err == nil {
				t.Errorf("Expected fillPDF() to return an error, but got nil")
			}
		})
	}
}

// TestExtractPages validates that the ATF and CLEO copies get the right pages and the CLEO copy never gets the SSN or serial number
func TestExtractPages(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	// Fill a form with just enough of Dr. Krieger's details to check for the SSN and serial number
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.SocialSecurityNumber = "078-05-1120"
	settings.UserData.Ethnicity = "Not Hispanic or Latino"
	settings.UserData.Race = []string{"White"}
	answers := rpqAnswers{ATFForm: rpqATFFormOptions[1], SerialNumber: "TGB126MS0001"}
	filledPDF, err := fillPDF(rpqTemplate, buildRPQFields(&settings, answers))
	if err != nil {
		t.Fatalf("fillPDF() returned an unexpected error: %v", err)
	}
	pdfConfig, err := pdfConfiguration()
	if err != nil {
		t.Fatalf("pdfConfiguration() returned an unexpected error: %v", err)
	}

	tests := []struct {
		name        string
		pageRange   string
		wantPages   int
		wantSSN     bool
		wantNameSet bool
	}{
		{"ATF copy", "1-4", 4, true, true},
		{"CLEO copy", "5-6", 2, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copyPDF, err := extractPages(filledPDF, test.pageRange)
			if err != nil {
				t.Fatalf("extractPages() returned an unexpected error: %v", err)
			}
			pageCount, err := api.PageCount(context.Background(), bytes.NewReader(copyPDF), pdfConfig)
			if err != nil {
				t.Fatalf("Failed to count pages: %v", err)
			}
			if pageCount != test.wantPages {
				t.Errorf("Expected %d pages, but got %d", test.wantPages, pageCount)
			}

			// Look through every text field in the copy for the SSN, serial number and Krieger's name
			exportedForms, err := api.ExportForm(context.Background(), bytes.NewReader(copyPDF), "test", pdfConfig)
			if err != nil {
				t.Fatalf("Failed to read the copy's form: %v", err)
			}
			foundSSN, foundName := false, false
			for _, exportedForm := range exportedForms.Forms {
				for _, textField := range exportedForm.TextFields {
					foundSSN = foundSSN || textField.Value == "078-05-1120" || textField.Value == "TGB126MS0001"
					foundName = foundName || strings.Contains(textField.Value, "Algernop Krieger")
				}
			}
			if foundSSN != test.wantSSN {
				t.Errorf("Expected SSN and serial number found to be '%t', but got '%t'", test.wantSSN, foundSSN)
			}
			if foundName != test.wantNameSet {
				t.Errorf("Expected Krieger's name found to be '%t', but got '%t'", test.wantNameSet, foundName)
			}
		})
	}
}

// TestWritePDF validates that a PDF is written with mode 0600 and no .partial file is left behind
func TestWritePDF(t *testing.T) {
	destinationPath := filepath.Join(t.TempDir(), "TGB126MS0001_TheKriegerCloneTrust_Krieger_RPQ_ATF.pdf")
	if err := writePDF(destinationPath, []byte("%PDF-1.7 Do you want ants?")); err != nil {
		t.Fatalf("writePDF() returned an unexpected error: %v", err)
	}

	outputFile, err := os.Stat(destinationPath)
	if err != nil {
		t.Fatalf("Expected %s to exist, but got: %v", destinationPath, err)
	}
	if outputFile.Mode().Perm() != 0600 {
		t.Errorf("Expected mode to be '%#o', but got '%#o'", 0600, outputFile.Mode().Perm())
	}
	if _, err := os.Stat(destinationPath + ".partial"); err == nil {
		t.Errorf("Expected %s to be gone, but it still exists", destinationPath+".partial")
	}
}

// TestFileNamePart validates that whitespace and path separators are stripped from file name parts
func TestFileNamePart(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"Spaces", "The Krieger Clone Trust", "TheKriegerCloneTrust"},
		{"Tabs and doubled spaces", "The  Krieger\tClone Trust ", "TheKriegerCloneTrust"},
		{"Path separators", "Figgis/Agency\\Trust", "FiggisAgencyTrust"},
		{"Already clean", "TGB126MS0001", "TGB126MS0001"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fileNamePart(test.value); got != test.want {
				t.Errorf("Expected file name part to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestLastName validates that the last name skips suffixes like Jr. and III
func TestLastName(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
	}{
		{"First and last", "Algernop Krieger", "Krieger"},
		{"Middle name", "Sterling Malory Archer", "Archer"},
		{"Jr. with a comma", "Cyril Figgis, Jr.", "Figgis"},
		{"Roman numeral", "Ray Gillette III", "Gillette"},
		{"Lowercase sr", "Len Trexler sr", "Trexler"},
		{"One name", "Pam", "Pam"},
		{"Only a suffix", "Jr.", ""},
		{"Blank", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := lastName(test.fullName); got != test.want {
				t.Errorf("Expected last name to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestReadFieldWidths validates that field widths are read from the template under their full names
func TestReadFieldWidths(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}
	// The Caliber column is the narrowest at about 35pt and Item 9 runs across the page at about 560pt
	tests := []struct {
		fieldName string
		minWidth  float64
		maxWidth  float64
	}{
		{interstatePage + "TextField81[0]", 34, 36},
		{interstatePage + "TextField16[0]", 559, 562},
	}
	for _, test := range tests {
		width, exists := fieldWidths[test.fieldName]
		if !exists {
			t.Errorf("Expected a width for %s, but there wasn't one", test.fieldName)
			continue
		}
		if width < test.minWidth || width > test.maxWidth {
			t.Errorf("Expected %s to be %.0f-%.0fpt wide, but got %.1f", test.fieldName, test.minWidth, test.maxWidth, width)
		}
	}
}

// TestTextFitsWidth validates text measurement against the 5320.20's 113pt Manufacturer column
func TestTextFitsWidth(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	tests := []struct {
		name     string
		text     string
		fontSize int
		want     bool
	}{
		{"Short name at 9pt", "Krieger Laboratories", 9, true},
		{"Company name at 9pt", "Mad Scientist Research LLC", 9, false},
		{"Company name at 8pt", "Mad Scientist Research LLC", 8, true},
		{"Spelled out at 6pt", "Mad Scientist Research Limited Liability Company", 6, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := textFitsWidth(test.text, 112.9, test.fontSize)
			if err != nil {
				t.Fatalf("textFitsWidth() returned an unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("Expected fits to be '%t', but got '%t'", test.want, got)
			}
		})
	}
}

// TestFitFontSizes validates that a value too wide at 9pt gets a smaller font, a short one keeps 9pt,
// and one that won't fit even at the smallest size is an error
func TestFitFontSizes(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()

	manufacturerField := interstatePage + "TextField51[0]"
	modelField := interstatePage + "TextField91[0]"
	fittedPDF, err := fitFontSizes(interstateTemplate, map[string]string{
		manufacturerField: "Mad Scientist Research LLC",
		modelField:        "The Gauss",
	})
	if err != nil {
		t.Fatalf("fitFontSizes() returned an unexpected error: %v", err)
	}

	// Read the font settings back out of the fitted template
	pdfContext, err := readFormContext(fittedPDF)
	if err != nil {
		t.Fatalf("readFormContext() returned an unexpected error: %v", err)
	}
	fontSettings := map[string]string{}
	err = visitFormFields(pdfContext.XRefTable, func(fullName string, fieldDict types.Dict) error {
		if fullName == manufacturerField || fullName == modelField {
			defaultAppearance, err := pdfContext.DereferenceStringOrHexLiteral(fieldDict["DA"], model.V10, nil)
			if err != nil {
				return err
			}
			fontSettings[fullName] = defaultAppearance
		}
		return nil
	})
	if err != nil {
		t.Fatalf("visitFormFields() returned an unexpected error: %v", err)
	}
	if !strings.Contains(fontSettings[manufacturerField], " 8 Tf") {
		t.Errorf("Expected the manufacturer font to be '%s', but got '%s'", "8 Tf", fontSettings[manufacturerField])
	}
	if !strings.Contains(fontSettings[modelField], " 9 Tf") {
		t.Errorf("Expected the model font to be '%s', but got '%s'", "9 Tf", fontSettings[modelField])
	}

	// A value that can't fit at any size is refused
	_, err = fitFontSizes(interstateTemplate, map[string]string{
		manufacturerField: "Mad Scientist Research Limited Liability Company",
	})
	if err == nil {
		t.Errorf("Expected fitFontSizes() to return an error, but got nil")
	}
}
