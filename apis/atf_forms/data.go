/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bufio"
	_ "embed"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
)

// ATFForm An ATF form that Forms Filler knows how to fill out
type ATFForm struct {
	Name          string
	CheckSettings func(settings *internal_funcs.FFConfig) error
	Fill          func(reader *bufio.Reader, settings *internal_funcs.FFConfig) error
}

// formFields The values to put into a PDF form, keyed by each field's full name
type formFields struct {
	TextFields   map[string]string
	CheckedBoxes []string
}

// rpqATFFormOption One of the ATF forms a 5320.23 can be submitted with, and its Item 1 checkbox
type rpqATFFormOption struct {
	Label     string
	FieldName string
}

// rpqAnswers The details unique to each 5320.23, collected at the prompts
type rpqAnswers struct {
	ATFForm      rpqATFFormOption
	FirearmType  string
	MakerName    string
	MakerAddress string
	Model        string
	Caliber      string
	SerialNumber string
}

// rpqCopy The pages and field names that differ between the two copies of the 5320.23
type rpqCopy struct {
	Name             string
	Pages            string
	FrontPage        string
	BackPage         string
	DateOfBirthField string
	HasSSNAndSerial  bool
}

// rpqTemplate The blank ATF Form 5320.23, built into the binary so it never has to be downloaded
//
//go:embed eFormsTemplates/f_5320.23_national_firearms_act_nfa_responsible_person_questionnaire.pdf
var rpqTemplate []byte

// pdfcpuConfigRoot Where pdfcpu keeps its config and the font it needs to fill form fields, next to the settings file
var pdfcpuConfigRoot, _ = internal_funcs.OutputUserPath(".config/scienceguns")

// multilineFieldFlag The field flag (bit 13 of Ff) that makes a PDF text field multi-line
var multilineFieldFlag = 1 << 12

// nameSuffixes Name suffixes skipped when finding a last name for a file name
var nameSuffixes = []string{"jr", "sr", "ii", "iii", "iv", "v"}

// supportedForms Every form Forms Filler can fill out, in the order they're shown in the picker
var supportedForms = []ATFForm{
	{
		Name:          "5320.23 - Responsible Persons Questionnaire",
		CheckSettings: checkRPQSettings,
		Fill:          fillRPQ,
	},
}

// rpqATFFormOptions The choices for Item 1, the ATF form the 5320.23 is submitted with
var rpqATFFormOptions = []rpqATFFormOption{
	{Label: "ATF Form 1 - Application to Make and Register a Firearm", FieldName: "form1[0]"},
	{Label: "ATF Form 4 - Application for Tax Paid Transfer and Registration of Firearm", FieldName: "form4[0]"},
	{Label: "ATF Form 5 - Application for Tax Exempt Transfer and Registration of Firearm", FieldName: "form5[0]"},
}

// rpqFirearmTypes The NFA firearm types from Definition 5 of the 5320.23, offered for Item 4a
var rpqFirearmTypes = []string{
	"Short-Barreled Rifle",
	"Short-Barreled Shotgun",
	"Silencer",
	"Machinegun",
	"Any Other Weapon",
	"Destructive Device",
}

// rpqCopies The ATF copy (pages 1-4) and the CLEO copy (pages 5-6) of the 5320.23.
// The CLEO copy has no SSN or serial number fields, and the date of birth field is numbered differently on each.
var rpqCopies = []rpqCopy{
	{
		Name:             "ATF",
		Pages:            "1-4",
		FrontPage:        "topmostSubform[0].Page1[0].",
		BackPage:         "topmostSubform[0].Page2[0].",
		DateOfBirthField: "#field[24]",
		HasSSNAndSerial:  true,
	},
	{
		Name:             "CLEO",
		Pages:            "5-6",
		FrontPage:        "topmostSubform[0].Page5[0].",
		BackPage:         "topmostSubform[0].Page6[0].",
		DateOfBirthField: "#field[22]",
		HasSSNAndSerial:  false,
	},
}

// rpqEthnicityBoxes The Item 3h checkbox for each ethnicity
var rpqEthnicityBoxes = map[string]string{
	"Hispanic or Latino":     "ehl[0]",
	"Not Hispanic or Latino": "nhl[0]",
}

// rpqRaceBoxes The Item 3i checkbox for each race
var rpqRaceBoxes = map[string]string{
	"American Indian or Alaska Native":          "aian[0]",
	"Asian":                                     "a[0]",
	"Black or African American":                 "baa[0]",
	"Native Hawaiian or Other Pacific Islander": "nhopi[0]",
	"White": "w[0]",
}

// rpqBackPageBoxes The Items 6 and 8 checkboxes that are the same on every 5320.23 this tool fills out.
// Forms Filler assumes the responsible person can pass NICS, so every Item 6 question is answered No,
// except 6.m.2 which only applies after a Yes on 6.m.1 and is answered N/A.
var rpqBackPageBoxes = []string{
	"CheckBoxno6a[0]", // 6.a Intends to make a firearm for a prohibited person
	"CheckBoxno6b[0]", // 6.b Intends to dispose of a firearm in furtherance of a felony
	"CheckBoxno1[0]",  // 6.c Under indictment or information
	"CheckBoxno2[0]",  // 6.d Convicted of a felony
	"CheckBoxno3[0]",  // 6.e Fugitive from justice
	"CheckBoxno4[0]",  // 6.f Unlawful user of a controlled substance
	"CheckBoxno5[0]",  // 6.g Adjudicated mental defective or committed
	"CheckBoxno6[0]",  // 6.h Dishonorable discharge
	"CheckBoxno7[0]",  // 6.i Subject to a restraining order
	"CheckBoxno8[0]",  // 6.j Misdemeanor crime of domestic violence
	"CheckBoxno9[0]",  // 6.k Renounced U.S. citizenship
	"CheckBoxno10[0]", // 6.l Alien illegally in the U.S.
	"CheckBoxno11[0]", // 6.m.1 Alien admitted under a nonimmigrant visa
	"CheckBoxNA[0]",   // 6.m.2 Exception to 6.m.1
	"no17[0]",         // 8 Has been issued a UPIN
}
