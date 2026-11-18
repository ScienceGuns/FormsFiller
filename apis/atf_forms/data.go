/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bufio"
	_ "embed"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"regexp"
	"time"
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

// interstateFirearm One firearm in Item 4 of the 5320.20
type interstateFirearm struct {
	Manufacturer  string
	FirearmType   string
	Caliber       string
	Model         string
	BarrelLength  string
	OverallLength string
	SerialNumber  string
}

// interstateAnswers The details of one 5320.20, from the settings file or the prompts
type interstateAnswers struct {
	RegisteredInNFRTR          bool
	ReturnToOriginalLocation   bool
	StartDate                  time.Time
	EndDate                    time.Time
	Firearms                   []interstateFirearm
	ReturnedFormMailingAddress string
	Reason                     string
	TransportingFrom           string
	TransportingFromCounty     string
	TransportingTo             string
	TransportingToCounty       string
	ModeOfTransportation       string
}

// interstateTemplate The blank ATF Form 5320.20, built into the binary so it never has to be downloaded
//
//go:embed eFormsTemplates/f_5320_20.pdf
var interstateTemplate []byte

// rpqTemplate The blank ATF Form 5320.23, built into the binary so it never has to be downloaded
//
//go:embed eFormsTemplates/f_5320_23.pdf
var rpqTemplate []byte

// now is a variable that can be overridden for testing so checks against today's date are predictable
var now = time.Now

// pdfcpuConfigRoot Where pdfcpu keeps its config and the font it needs to fill form fields, next to the settings file
var pdfcpuConfigRoot, _ = internal_funcs.OutputUserPath(".config/scienceguns")

// multilineFieldFlag The field flag (bit 13 of Ff) that makes a PDF text field multi-line
var multilineFieldFlag = 1 << 12

// fontSizePattern A field's default appearance split into its font name, font size and anything after "Tf"
var fontSizePattern = regexp.MustCompile(`^(/\S+)\s+([\d.]+)\s+Tf(.*)$`)

// minimumFontSize The smallest font size a value is shrunk to before it's rejected as too long for its field
var minimumFontSize = 6

// fieldPadding The space in points pdfcpu leaves between a field's edge and its text, on each side
var fieldPadding = 2.0

// fillFontName The font pdfcpu draws filled-in values with
var fillFontName = "Roboto-Regular"

// nameSuffixes Name suffixes skipped when finding a last name for a file name
var nameSuffixes = []string{"jr", "sr", "ii", "iii", "iv", "v"}

// supportedForms Every form Forms Filler can fill out, in the order they're shown in the picker
var supportedForms = []ATFForm{
	{
		Name:          "5320.23 - Responsible Persons Questionnaire",
		CheckSettings: checkRPQSettings,
		Fill:          fillRPQ,
	},
	{
		Name:          "5320.20 - Application to Transport Interstate",
		CheckSettings: checkInterstateSettings,
		Fill:          fillInterstate,
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

// interstatePage The prefix of every field name on the 5320.20, which only has fields on page 1
var interstatePage = "topmostSubform[0].Page1[0]."

// interstateMaxFirearms The number of rows in Item 4 of the 5320.20. More firearms need another form.
var interstateMaxFirearms = 3

// interstateFirearmColumns The Item 4 field name for each column, with the row number (1-3) added before "[0]"
var interstateFirearmColumns = struct {
	Manufacturer  string
	FirearmType   string
	Caliber       string
	Model         string
	BarrelLength  string
	OverallLength string
	SerialNumber  string
}{
	Manufacturer:  "TextField5%d[0]",
	FirearmType:   "TextField6%d[0]",
	Caliber:       "TextField8%d[0]",
	Model:         "TextField9%d[0]",
	BarrelLength:  "TextField10%d[0]",
	OverallLength: "TextField11%d[0]",
	SerialNumber:  "TextField12%d[0]",
}

// interstateFirearmTypes The NFA firearm types that need a 5320.20 to cross state lines, offered for Item 4b.
// Silencers and AOWs don't need one.
var interstateFirearmTypes = []string{
	"Destructive Device",
	"Machinegun",
	"Short-Barreled Rifle",
	"Short-Barreled Shotgun",
}

// interstateUnregisteredFirearmTypes The only types ATF accepts on a 5320.20 for firearms not registered to the owner in the NFRTR
var interstateUnregisteredFirearmTypes = []string{
	"Short-Barreled Rifle",
	"Short-Barreled Shotgun",
}

// caliberPattern Calibers are whole numbers on ATF forms, like 556 or 300
var caliberPattern = regexp.MustCompile(`^\d+$`)

// lengthPattern Barrel and overall lengths are numbers of inches, like 10 or 10.5, with no unit
var lengthPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)

// formDatePattern The MM/DD/YYYY format with leading zeroes that ATF forms use
var formDatePattern = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)

// formDateLayout The Go time layout matching formDatePattern
var formDateLayout = "01/02/2006"

// fileNameDateLayout The YYYY-MM-DD date layout used in file names so they sort by date
var fileNameDateLayout = "2006-01-02"

// countyNameEndings Words that already end a county's name, so " County" isn't added after them
var countyNameEndings = []string{"county", "parish", "borough"}
