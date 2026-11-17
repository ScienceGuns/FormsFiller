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
	"time"
)

// TestCheckInterstateSettings validates that the 5320.20 is available with or without a trust
func TestCheckInterstateSettings(t *testing.T) {
	tests := []struct {
		name      string
		trustName string
	}{
		{"With a trust", "The Krieger Clone Trust"},
		{"Without a trust", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var settings internal_funcs.FFConfig
			settings.UserData.TrustName = test.trustName
			if err := checkInterstateSettings(&settings); err != nil {
				t.Errorf("checkInterstateSettings() error = %v, wantErr %v", err, false)
			}
		})
	}
}

// TestPromptInterstateAnswers validates every 5320.20 prompt when the settings file has none of the answers
func TestPromptInterstateAnswers(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	// Set a fixed point in time (Archer's original air date) so the trip's dates aren't in the past
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}

	// Dr. Krieger takes The Gauss to Texas and back, fumbling a few answers along the way
	var settings internal_funcs.FFConfig
	input := strings.Join([]string{
		"",                                  // Registered in the NFRTR (default yes)
		"",                                  // Item 2 returning (default yes)
		"09/17/2009",                        // Item 3 leaving
		"09/16/2009",                        // Item 3 returning before leaving is rejected
		"09/20/2009",                        // Item 3 returning
		"Mad Scientist Research LLC",        // Item 4a
		"3",                                 // Item 4b Short-Barreled Rifle
		"5.56 NATO",                         // Item 4c a name is rejected
		"300",                               // Item 4c
		"The Gauss",                         // Item 4d
		"10.5\"",                            // Item 4e a unit is rejected
		"10.5",                              // Item 4e
		"27.25",                             // Item 4f
		"TGB126MS0001",                      // Item 4g
		"",                                  // Another firearm (default no)
		"",                                  // Item 5 blank
		"Training at the Danger Zone range", // Item 6
		"1 Figgis Agency Plaza, New York, NY 10001", // Item 7
		"New York",                            // Item 7 county
		"934 Ranch Road, Galveston, TX 77550", // Item 8
		"Galveston",                           // Item 8 county
		"Personal vehicle",                    // Item 9
	}, "\n") + "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	answers, err := promptInterstateAnswers(reader, &settings, fieldWidths)
	if err != nil {
		t.Fatalf("promptInterstateAnswers() returned an unexpected error: %v", err)
	}
	if leftover, _ := reader.ReadString('\n'); leftover != "" {
		t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
	}
	if !answers.RegisteredInNFRTR || !answers.ReturnToOriginalLocation {
		t.Errorf("Expected registered and returning to be 'true', but got '%t' and '%t'", answers.RegisteredInNFRTR, answers.ReturnToOriginalLocation)
	}
	if !answers.StartDate.Equal(time.Date(2009, time.September, 17, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Expected start date to be '%s', but got '%s'", "09/17/2009", answers.StartDate.Format(formDateLayout))
	}
	if !answers.EndDate.Equal(time.Date(2009, time.September, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Expected end date to be '%s', but got '%s'", "09/20/2009", answers.EndDate.Format(formDateLayout))
	}
	if len(answers.Firearms) != 1 {
		t.Fatalf("Expected %d firearm, but got %d", 1, len(answers.Firearms))
	}
	wantFirearm := interstateFirearm{
		Manufacturer:  "Mad Scientist Research LLC",
		FirearmType:   "Short-Barreled Rifle",
		Caliber:       "300",
		Model:         "The Gauss",
		BarrelLength:  "10.5",
		OverallLength: "27.25",
		SerialNumber:  "TGB126MS0001",
	}
	if answers.Firearms[0] != wantFirearm {
		t.Errorf("Expected firearm to be '%+v', but got '%+v'", wantFirearm, answers.Firearms[0])
	}
	checks := []struct {
		field string
		want  string
		got   string
	}{
		{"returned form address", "", answers.ReturnedFormMailingAddress},
		{"reason", "Training at the Danger Zone range", answers.Reason},
		{"transporting from", "1 Figgis Agency Plaza, New York, NY 10001", answers.TransportingFrom},
		{"transporting from county", "New York", answers.TransportingFromCounty},
		{"transporting to", "934 Ranch Road, Galveston, TX 77550", answers.TransportingTo},
		{"transporting to county", "Galveston", answers.TransportingToCounty},
		{"mode of transportation", "Personal vehicle", answers.ModeOfTransportation},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("Expected %s to be '%s', but got '%s'", check.field, check.want, check.got)
		}
	}
}

// TestPromptInterstateAnswers_FromSettings validates that answers in the settings file aren't asked for,
// and that unregistered firearms are limited to SBRs and SBSs
func TestPromptInterstateAnswers_FromSettings(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	// Set a fixed point in time (Archer's original air date) so the trip's dates aren't in the past
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}

	// Dr. Krieger's settings answer Items 2, 5, 6, 7 and 9 for a permanent move
	var settings internal_funcs.FFConfig
	returning := false
	noMailingAddress := ""
	settings.UserData.County = "New York"
	settings.Interstate.ReturnToOriginalLocation = &returning
	settings.Interstate.ReturnedFormMailingAddress = &noMailingAddress
	settings.Interstate.Reason = "Permanent change of address"
	settings.Interstate.TransportingFrom = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.Interstate.ModeOfTransportation = "Personal vehicle"

	// Only the NFRTR question, Items 3, 4 and 8 are asked. Unregistered firearms only offer SBR (1) and SBS (2),
	// so 3 is rejected.
	input := strings.Join([]string{
		"n", "09/17/2009", "09/18/2009",
		"Mad Scientist Research LLC", "3", "1", "300", "The Gauss", "10.5", "27.25", "TGB126MS0001", "",
		"934 Ranch Road, Galveston, TX 77550", "Galveston County",
	}, "\n") + "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	answers, err := promptInterstateAnswers(reader, &settings, fieldWidths)
	if err != nil {
		t.Fatalf("promptInterstateAnswers() returned an unexpected error: %v", err)
	}
	if leftover, _ := reader.ReadString('\n'); leftover != "" {
		t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
	}
	if answers.RegisteredInNFRTR || answers.ReturnToOriginalLocation {
		t.Errorf("Expected registered and returning to be 'false', but got '%t' and '%t'", answers.RegisteredInNFRTR, answers.ReturnToOriginalLocation)
	}
	if answers.Firearms[0].FirearmType != "Short-Barreled Rifle" {
		t.Errorf("Expected firearm type to be '%s', but got '%s'", "Short-Barreled Rifle", answers.Firearms[0].FirearmType)
	}
	checks := []struct {
		field string
		want  string
		got   string
	}{
		{"reason", "Permanent change of address", answers.Reason},
		{"transporting from", "1 Figgis Agency Plaza, New York, NY 10001", answers.TransportingFrom},
		{"transporting from county", "New York", answers.TransportingFromCounty},
		{"transporting to county", "Galveston County", answers.TransportingToCounty},
		{"mode of transportation", "Personal vehicle", answers.ModeOfTransportation},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("Expected %s to be '%s', but got '%s'", check.field, check.want, check.got)
		}
	}
}

// TestPromptInterstateAnswers_ThreeFirearms validates that Item 4 stops at three firearms without asking for another
func TestPromptInterstateAnswers_ThreeFirearms(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	// Set a fixed point in time (Archer's original air date) so the trip's dates aren't in the past
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}

	// Three Gauss SBRs, with a yes after the first two and no question after the third
	var settings internal_funcs.FFConfig
	gauss := func(serialNumber string) []string {
		return []string{"Mad Scientist Research LLC", "3", "300", "The Gauss", "10.5", "27.25", serialNumber}
	}
	lines := []string{"", "", "09/17/2009", "09/20/2009"}
	lines = append(lines, gauss("TGB126MS0001")...)
	lines = append(lines, "y")
	lines = append(lines, gauss("TGB126MS0002")...)
	lines = append(lines, "y")
	lines = append(lines, gauss("TGB126MS0003")...)
	lines = append(lines, "", "Training", "1 Figgis Agency Plaza, New York, NY 10001", "New York",
		"934 Ranch Road, Galveston, TX 77550", "Galveston", "Personal vehicle")
	reader := bufio.NewReader(strings.NewReader(strings.Join(lines, "\n") + "\n"))

	answers, err := promptInterstateAnswers(reader, &settings, fieldWidths)
	if err != nil {
		t.Fatalf("promptInterstateAnswers() returned an unexpected error: %v", err)
	}
	if leftover, _ := reader.ReadString('\n'); leftover != "" {
		t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
	}
	if len(answers.Firearms) != 3 {
		t.Fatalf("Expected %d firearms, but got %d", 3, len(answers.Firearms))
	}
	if answers.Firearms[2].SerialNumber != "TGB126MS0003" {
		t.Errorf("Expected third serial number to be '%s', but got '%s'", "TGB126MS0003", answers.Firearms[2].SerialNumber)
	}
}

// TestPromptInterstateFirearm_TooLong validates that a value too long for its column even at the smallest font is asked for again
func TestPromptInterstateFirearm_TooLong(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}

	// The spelled out company name won't fit in the Manufacturer column, so the short one is used
	input := "Mad Scientist Research Limited Liability Company\nMad Scientist Research LLC\n3\n300\nThe Gauss\n10.5\n27.25\nTGB126MS0001\n"
	reader := bufio.NewReader(strings.NewReader(input))
	firearm, err := promptInterstateFirearm(reader, 1, interstateFirearmTypes, fieldWidths)
	if err != nil {
		t.Fatalf("promptInterstateFirearm() returned an unexpected error: %v", err)
	}
	if firearm.Manufacturer != "Mad Scientist Research LLC" {
		t.Errorf("Expected manufacturer to be '%s', but got '%s'", "Mad Scientist Research LLC", firearm.Manufacturer)
	}
}

// TestValidateCaliber validates that calibers are whole numbers the way ATF forms want them
func TestValidateCaliber(t *testing.T) {
	tests := []struct {
		name    string
		caliber string
		wantErr bool
	}{
		{"556", "556", false},
		{"300", "300", false},
		{"Gauge", "12", false},
		{"Decimal", "5.56", true},
		{"Name", "5.56 NATO", true},
		{"Leading dot", ".300", true},
		{"Cartridge name", "300 BLK", true},
		{"Blank", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCaliber(test.caliber)
			if (err != nil) != test.wantErr {
				t.Errorf("validateCaliber() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestValidateLength validates that lengths are plain numbers of inches
func TestValidateLength(t *testing.T) {
	tests := []struct {
		name    string
		length  string
		wantErr bool
	}{
		{"Whole number", "10", false},
		{"Decimal", "10.5", false},
		{"Two decimal places", "27.25", false},
		{"Inch mark", "8\"", true},
		{"Unit", "10 inches", true},
		{"Leading dot", ".5", true},
		{"Trailing dot", "10.", true},
		{"Blank", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateLength(test.length)
			if (err != nil) != test.wantErr {
				t.Errorf("validateLength() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// TestCountyName validates that " County" is only added when the name doesn't already end with a county word
func TestCountyName(t *testing.T) {
	tests := []struct {
		name   string
		county string
		want   string
	}{
		{"Bare name", "New York", "New York County"},
		{"Already a county", "Montgomery County", "Montgomery County"},
		{"Lowercase county", "galveston county", "galveston county"},
		{"Louisiana parish", "Orleans Parish", "Orleans Parish"},
		{"Alaska borough", "Matanuska-Susitna Borough", "Matanuska-Susitna Borough"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := countyName(test.county); got != test.want {
				t.Errorf("Expected county to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestInterstateFilePath validates the {TravelStartDate}_{Owner}_{ItemCount}Gun(s)_5320_20.pdf file name
func TestInterstateFilePath(t *testing.T) {
	tests := []struct {
		name      string
		trustName string
		firearms  int
		want      string
	}{
		{"Trust with one firearm", "The Krieger Clone Trust", 1, "/tmp/isis/2009-09-17_TheKriegerCloneTrust_1Gun_5320_20.pdf"},
		{"Individual with three firearms", "", 3, "/tmp/isis/2009-09-17_Krieger_3Guns_5320_20.pdf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var settings internal_funcs.FFConfig
			settings.Settings.OutputPath = "/tmp/isis"
			settings.UserData.TrustName = test.trustName
			settings.UserData.FullLegalName = "Algernop Krieger"
			answers := interstateAnswers{
				StartDate: time.Date(2009, time.September, 17, 0, 0, 0, 0, time.UTC),
				Firearms:  make([]interstateFirearm, test.firearms),
			}
			if got := interstateFilePath(&settings, answers); got != test.want {
				t.Errorf("Expected file path to be '%s', but got '%s'", test.want, got)
			}
		})
	}
}

// TestBuildInterstateFields validates the 5320.20 field values for Dr. Krieger's trust taking The Gauss to Texas and back
func TestBuildInterstateFields(t *testing.T) {
	// Dr. Krieger's settings
	var settings internal_funcs.FFConfig
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.PhoneNumber = "212-555-0187"
	settings.UserData.EmailAddress = "dr.krieger@figgis.example"
	answers := interstateAnswers{
		RegisteredInNFRTR:        true,
		ReturnToOriginalLocation: true,
		StartDate:                time.Date(2009, time.September, 17, 0, 0, 0, 0, time.UTC),
		EndDate:                  time.Date(2009, time.September, 20, 0, 0, 0, 0, time.UTC),
		Firearms: []interstateFirearm{
			{"Mad Scientist Research LLC", "Short-Barreled Rifle", "300", "The Gauss", "10.5", "27.25", "TGB126MS0001"},
			{"Mad Scientist Research LLC", "Short-Barreled Rifle", "300", "The Gauss", "10.5", "27.25", "TGB126MS0002"},
		},
		Reason:                 "Training at the Danger Zone range",
		TransportingFrom:       "1 Figgis Agency Plaza, New York, NY 10001",
		TransportingFromCounty: "New York",
		TransportingTo:         "934 Ranch Road, Galveston, TX 77550",
		TransportingToCounty:   "Galveston County",
		ModeOfTransportation:   "Personal vehicle",
	}

	fields := buildInterstateFields(&settings, answers)

	page := interstatePage
	wantText := map[string]string{
		page + "TextField2[0]":           "The Krieger Clone Trust\n1 Figgis Agency Plaza, New York, NY 10001",
		page + "TextField3[0]":           "212-555-0187",
		page + "TextField4[0]":           "dr.krieger@figgis.example",
		page + "TextField4[1]":           "09/17/2009",
		page + "TextField4to[0]":         "09/20/2009",
		page + "TextField51[0]":          "Mad Scientist Research LLC",
		page + "TextField61[0]":          "Short-Barreled Rifle",
		page + "TextField81[0]":          "300",
		page + "TextField91[0]":          "The Gauss",
		page + "TextField101[0]":         "10.5",
		page + "TextField111[0]":         "27.25",
		page + "TextField121[0]":         "TGB126MS0001",
		page + "TextField122[0]":         "TGB126MS0002",
		page + "TextField13[0]":          "Training at the Danger Zone range",
		page + "TextField15[0]":          "1 Figgis Agency Plaza, New York, NY 10001\nNew York County",
		page + "TextField15transport[0]": "934 Ranch Road, Galveston, TX 77550\nGalveston County",
		page + "TextField16[0]":          "Personal vehicle",
	}
	for fieldName, want := range wantText {
		if got := fields.TextFields[fieldName]; got != want {
			t.Errorf("Expected %s to be '%s', but got '%s'", fieldName, want, got)
		}
	}
	// Item 5 is left out entirely when it's blank, and there's no third firearm
	for _, fieldName := range []string{page + "TextField124[0]", page + "TextField123[0]"} {
		if _, exists := fields.TextFields[fieldName]; exists {
			t.Errorf("Expected %s to not be filled, but it was", fieldName)
		}
	}
	for _, fieldName := range []string{page + "tru[0]", page + "yes[0]"} {
		if !slices.Contains(fields.CheckedBoxes, fieldName) {
			t.Errorf("Expected %s to be checked, but it wasn't", fieldName)
		}
	}
	for _, fieldName := range []string{page + "ind[0]", page + "no[0]"} {
		if slices.Contains(fields.CheckedBoxes, fieldName) {
			t.Errorf("Expected %s to not be checked, but it was", fieldName)
		}
	}
}

// TestBuildInterstateFields_Individual validates that without a trust the individual is the owner, and a one-way trip checks No
func TestBuildInterstateFields_Individual(t *testing.T) {
	var settings internal_funcs.FFConfig
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.FullLegalAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	answers := interstateAnswers{ReturnedFormMailingAddress: "Krieger's Van, 1 Figgis Agency Plaza, New York, NY 10001"}

	fields := buildInterstateFields(&settings, answers)

	page := interstatePage
	if got := fields.TextFields[page+"TextField2[0]"]; got != "Algernop Krieger\n1 Figgis Agency Plaza, New York, NY 10001" {
		t.Errorf("Expected owner to be '%s', but got '%s'", "Algernop Krieger\n1 Figgis Agency Plaza, New York, NY 10001", got)
	}
	if got := fields.TextFields[page+"TextField124[0]"]; got != answers.ReturnedFormMailingAddress {
		t.Errorf("Expected returned form address to be '%s', but got '%s'", answers.ReturnedFormMailingAddress, got)
	}
	for _, fieldName := range []string{page + "ind[0]", page + "no[0]"} {
		if !slices.Contains(fields.CheckedBoxes, fieldName) {
			t.Errorf("Expected %s to be checked, but it wasn't", fieldName)
		}
	}
}

// TestFillInterstate validates a full run that saves Dr. Krieger's 5320.20, shrinking the manufacturer to fit
func TestFillInterstate(t *testing.T) {
	// Keep pdfcpu's config and the finished form in temp directories
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	// Set a fixed point in time (Archer's original air date) so the trip's dates aren't in the past
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()
	var settings internal_funcs.FFConfig
	settings.Settings.OutputPath = t.TempDir()
	settings.UserData.TrustName = "The Krieger Clone Trust"
	settings.UserData.TrustAddress = "1 Figgis Agency Plaza, New York, NY 10001"
	settings.UserData.FullLegalName = "Algernop Krieger"
	settings.UserData.PhoneNumber = "212-555-0187"
	settings.UserData.EmailAddress = "dr.krieger@figgis.example"

	// Answer everything, then confirm at the review with Enter
	input := strings.Join([]string{
		"", "", "09/17/2009", "09/20/2009",
		"Mad Scientist Research LLC", "3", "300", "The Gauss", "10.5", "27.25", "TGB126MS0001", "",
		"", "Training at the Danger Zone range", "1 Figgis Agency Plaza, New York, NY 10001", "New York",
		"934 Ranch Road, Galveston, TX 77550", "Galveston", "Personal vehicle", "",
	}, "\n") + "\n"
	reader := bufio.NewReader(strings.NewReader(input))
	if err := fillInterstate(reader, &settings); err != nil {
		t.Fatalf("fillInterstate() returned an unexpected error: %v", err)
	}
	if leftover, _ := reader.ReadString('\n'); leftover != "" {
		t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
	}

	// The form is saved under its expected name, readable only by the owner
	outputPath := filepath.Join(settings.Settings.OutputPath, "2009-09-17_TheKriegerCloneTrust_1Gun_5320_20.pdf")
	outputFile, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Expected %s to exist, but got: %v", outputPath, err)
	}
	if outputFile.Mode().Perm() != 0600 {
		t.Errorf("Expected mode to be '%#o', but got '%#o'", 0600, outputFile.Mode().Perm())
	}
}

// TestPromptInterstateAnswers_DefaultFrom validates that Item 7 offers the owner's address and the settings file's county
// as defaults when the settings file has no transporting-from address
func TestPromptInterstateAnswers_DefaultFrom(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	// Set a fixed point in time (Archer's original air date) so the trip's dates aren't in the past
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}

	tests := []struct {
		name         string
		trustAddress string
		countyInput  string
		wantFrom     string
		wantCounty   string
	}{
		{"Trust address and the settings county", "934 Ranch Road, Galveston, TX 77550", "", "934 Ranch Road, Galveston, TX 77550", "New York"},
		{"Individual's address and a typed county", "", "Montgomery", "1 Figgis Agency Plaza, New York, NY 10001", "Montgomery"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// The settings have a county, but no transporting-from address for it to go with
			var settings internal_funcs.FFConfig
			settings.UserData.TrustAddress = test.trustAddress
			settings.UserData.FullLegalAddress = "1 Figgis Agency Plaza, New York, NY 10001"
			settings.UserData.County = "New York"

			// Press Enter at the Item 7 address to take the default, then take or replace the default county
			input := strings.Join([]string{
				"", "", "09/17/2009", "09/20/2009",
				"Mad Scientist Research LLC", "3", "300", "The Gauss", "10.5", "27.25", "TGB126MS0001", "",
				"", "Training at the Danger Zone range", "", test.countyInput,
				"Perkiomenville, PA 18074", "Montgomery", "Personal vehicle",
			}, "\n") + "\n"
			reader := bufio.NewReader(strings.NewReader(input))

			answers, err := promptInterstateAnswers(reader, &settings, fieldWidths)
			if err != nil {
				t.Fatalf("promptInterstateAnswers() returned an unexpected error: %v", err)
			}
			if leftover, _ := reader.ReadString('\n'); leftover != "" {
				t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
			}
			if answers.TransportingFrom != test.wantFrom {
				t.Errorf("Expected transporting from to be '%s', but got '%s'", test.wantFrom, answers.TransportingFrom)
			}
			if answers.TransportingFromCounty != test.wantCounty {
				t.Errorf("Expected transporting from county to be '%s', but got '%s'", test.wantCounty, answers.TransportingFromCounty)
			}
		})
	}
}

// TestPromptFormDate validates that Item 3 rejects badly formatted dates, dates before today, and end dates before the start
func TestPromptFormDate(t *testing.T) {
	airDate := time.Date(2009, time.September, 17, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		input     string
		notBefore time.Time
		want      time.Time
		wantErr   bool
	}{
		{"Today", "09/17/2009\n", airDate, airDate, false},
		{"Later", "09/20/2009\n", airDate, time.Date(2009, time.September, 20, 0, 0, 0, 0, time.UTC), false},
		{"Past then today", "09/16/2009\n09/17/2009\n", airDate, airDate, false},
		{"Bad format then today", "9/17/2009\n2009-09-17\n02/30/2009\n09/17/2009\n", airDate, airDate, false},
		{"Input ends on a past date", "09/16/2009\n", airDate, time.Time{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.input))
			got, err := promptFormDate(reader, "When does Archer leave?", test.notBefore, "today")
			if (err != nil) != test.wantErr {
				t.Fatalf("promptFormDate() error = %v, wantErr %v", err, test.wantErr)
			}
			if !got.Equal(test.want) {
				t.Errorf("Expected date to be '%s', but got '%s'", test.want.Format(formDateLayout), got.Format(formDateLayout))
			}
		})
	}
}

// TestPromptInterstateAnswers_PastDates validates that a leave date before today and a return date before the leave
// date are both asked for again
func TestPromptInterstateAnswers_PastDates(t *testing.T) {
	// Keep pdfcpu's config in a temp directory
	realPdfcpuConfigRoot := pdfcpuConfigRoot
	pdfcpuConfigRoot = t.TempDir()
	defer func() { pdfcpuConfigRoot = realPdfcpuConfigRoot }()
	// Set a fixed point in time (Archer's original air date) so the trip's dates aren't in the past
	realNow := now
	now = func() time.Time { return time.Date(2009, time.September, 17, 12, 0, 0, 0, time.UTC) }
	defer func() { now = realNow }()
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		t.Fatalf("readFieldWidths() returned an unexpected error: %v", err)
	}

	// Leaving the day before the air date is rejected, and so is returning before leaving
	var settings internal_funcs.FFConfig
	input := strings.Join([]string{
		"", "", "09/16/2009", "09/18/2009", "09/17/2009", "09/20/2009",
		"Mad Scientist Research LLC", "3", "300", "The Gauss", "10.5", "27.25", "TGB126MS0001", "",
		"", "Training at the Danger Zone range", "1 Figgis Agency Plaza, New York, NY 10001", "New York",
		"934 Ranch Road, Galveston, TX 77550", "Galveston", "Personal vehicle",
	}, "\n") + "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	answers, err := promptInterstateAnswers(reader, &settings, fieldWidths)
	if err != nil {
		t.Fatalf("promptInterstateAnswers() returned an unexpected error: %v", err)
	}
	if leftover, _ := reader.ReadString('\n'); leftover != "" {
		t.Errorf("Expected all input to be used, but '%s' was left over", leftover)
	}
	if answers.StartDate.Format(formDateLayout) != "09/18/2009" {
		t.Errorf("Expected start date to be '%s', but got '%s'", "09/18/2009", answers.StartDate.Format(formDateLayout))
	}
	if answers.EndDate.Format(formDateLayout) != "09/20/2009" {
		t.Errorf("Expected end date to be '%s', but got '%s'", "09/20/2009", answers.EndDate.Format(formDateLayout))
	}
}
