/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// checkInterstateSettings Check that the settings have what Form 5320.20 needs. Every valid settings file does,
// since the owner is the trust when there is one and the individual otherwise.
func checkInterstateSettings(settings *internal_funcs.FFConfig) error {
	return nil
}

// fillInterstate Guide the user through the details of a 5320.20, then save it
func fillInterstate(reader *bufio.Reader, settings *internal_funcs.FFConfig) error {
	// Some Item 4 columns are narrow, so answers are checked against each field's width before they're accepted
	fieldWidths, err := readFieldWidths(interstateTemplate)
	if err != nil {
		return err
	}
	// Measure once up front so a problem loading the font shows up as an error instead of a rejected answer
	if _, err := textFitsWidth("", 0, minimumFontSize); err != nil {
		return err
	}

	// Keep asking until the user confirms the answers look right
	var answers interstateAnswers
	for {
		answers, err = promptInterstateAnswers(reader, settings, fieldWidths)
		if err != nil {
			return err
		}
		printInterstateSummary(settings, answers)
		confirmed, err := internal_funcs.PromptYesNo(reader, "\nDoes this look correct?", true)
		if err != nil {
			return err
		}
		if confirmed {
			break
		}
		fmt.Println("\nLet's try that again.")
	}

	// Check for an earlier copy of this form before doing any work
	outputPath := interstateFilePath(settings, answers)
	if _, err := os.Stat(outputPath); err == nil {
		overwrite, err := internal_funcs.PromptYesNo(reader, fmt.Sprintf("\n%s already exists. Overwrite it?", outputPath), false)
		if err != nil {
			return err
		}
		if !overwrite {
			fmt.Println("Leaving the existing file alone. Nothing was saved.")
			return nil // This is not an error since the user chose to stop
		}
	}

	// Shrink any value that's too wide for its field at the form's font size, then fill the form
	fields := buildInterstateFields(settings, answers)
	fittedTemplate, err := fitFontSizes(interstateTemplate, fields.TextFields)
	if err != nil {
		return err
	}
	filledPDF, err := fillPDF(fittedTemplate, fields)
	if err != nil {
		return err
	}
	if err := writePDF(outputPath, filledPDF); err != nil {
		return err
	}
	fmt.Printf("\nSaved the 5320.20 to %s\n", outputPath)

	// Remind the user of everything Forms Filler leaves to them, including where ATF wants this form sent
	fmt.Println("\nBefore submitting:")
	fmt.Println("  1. Sign and date the form.")
	if answers.RegisteredInNFRTR {
		fmt.Println("  2. Submit it through eForms, or email it to nfafax@atf.gov.")
	} else {
		fmt.Println("  2. Email it to nfafax@atf.gov. eForms only accepts firearms registered to you in the NFRTR.")
	}
	fmt.Println("  3. If a common or contract carrier is moving the firearms, give the carrier a copy of the approved form.")
	return nil
}

// promptInterstateAnswers Collect the details of a 5320.20, asking only for what the settings file doesn't already have
func promptInterstateAnswers(reader *bufio.Reader, settings *internal_funcs.FFConfig, fieldWidths map[string]float64) (interstateAnswers, error) {
	var answers interstateAnswers
	var err error

	// ATF won't take registered and unregistered firearms on the same form, and only takes unregistered SBRs and SBSs
	answers.RegisteredInNFRTR, err = internal_funcs.PromptYesNo(reader, "\nAre these firearms registered to the owner in the NFRTR?", true)
	if err != nil {
		return interstateAnswers{}, err
	}
	firearmTypes := interstateFirearmTypes
	if !answers.RegisteredInNFRTR {
		firearmTypes = interstateUnregisteredFirearmTypes
		fmt.Println("Only short-barreled rifles and shotguns can go on a 5320.20 for firearms not registered to you.")
	}

	// Item 2 - Whether the firearms come back
	if settings.Interstate.ReturnToOriginalLocation != nil {
		answers.ReturnToOriginalLocation = *settings.Interstate.ReturnToOriginalLocation
	} else {
		answers.ReturnToOriginalLocation, err = internal_funcs.PromptYesNo(reader, "\nItem 2 - Will the firearms be returned to their original location?", true)
		if err != nil {
			return interstateAnswers{}, err
		}
	}

	// Item 3 - The dates away, or the dates in transit for a permanent move
	startQuestion, endQuestion := "Item 3 - Date the firearms leave (MM/DD/YYYY)", "Item 3 - Date the firearms return (MM/DD/YYYY)"
	if !answers.ReturnToOriginalLocation {
		startQuestion, endQuestion = "Item 3 - Date transport starts (MM/DD/YYYY)", "Item 3 - Date transport ends (MM/DD/YYYY)"
	}
	fmt.Println()
	currentTime := now()
	today := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
	answers.StartDate, err = promptFormDate(reader, startQuestion, today, "today")
	if err != nil {
		return interstateAnswers{}, err
	}
	answers.EndDate, err = promptFormDate(reader, endQuestion, answers.StartDate, "the start date")
	if err != nil {
		return interstateAnswers{}, err
	}

	// Item 4 - Up to three firearms, one row each
	for row := 1; row <= interstateMaxFirearms; row++ {
		firearm, err := promptInterstateFirearm(reader, row, firearmTypes, fieldWidths)
		if err != nil {
			return interstateAnswers{}, err
		}
		answers.Firearms = append(answers.Firearms, firearm)
		if row == interstateMaxFirearms {
			fmt.Printf("That fills all %d rows of Item 4. Any other firearms need their own 5320.20.\n", interstateMaxFirearms)
			break
		}
		another, err := internal_funcs.PromptYesNo(reader, "\nAdd another firearm?", false)
		if err != nil {
			return interstateAnswers{}, err
		}
		if !another {
			break
		}
	}

	// Item 5 - Usually blank, which means the approved form goes back to the Item 1 address
	fmt.Println()
	if settings.Interstate.ReturnedFormMailingAddress != nil {
		answers.ReturnedFormMailingAddress = *settings.Interstate.ReturnedFormMailingAddress
	} else {
		answers.ReturnedFormMailingAddress, err = internal_funcs.PromptTextOrBlank(reader, "Item 5 - Mailing address for the approved form (leave blank to use the Item 1 address)")
		if err != nil {
			return interstateAnswers{}, err
		}
	}

	// Item 6 - From the settings file when it's there
	answers.Reason = settings.Interstate.Reason
	if answers.Reason == "" {
		answers.Reason, err = internal_funcs.PromptRequiredText(reader, "Item 6 - Reason for transporting the firearms")
		if err != nil {
			return interstateAnswers{}, err
		}
	}

	// Item 7 - The settings file's address and county go together, since the county is for that address.
	// Without one, the owner's address and the settings file's county are offered as defaults.
	if settings.Interstate.TransportingFrom != "" {
		answers.TransportingFrom = settings.Interstate.TransportingFrom
		answers.TransportingFromCounty = settings.UserData.County
	} else {
		defaultFrom := settings.UserData.TrustAddress
		if defaultFrom == "" {
			defaultFrom = settings.UserData.FullLegalAddress
		}
		answers.TransportingFrom, err = internal_funcs.PromptTextWithDefault(reader, "Item 7 - Address the firearms are transported from", defaultFrom)
		if err != nil {
			return interstateAnswers{}, err
		}
	}
	if answers.TransportingFromCounty == "" {
		answers.TransportingFromCounty, err = internal_funcs.PromptTextWithDefault(reader, "Item 7 - County the firearms are transported from", settings.UserData.County)
		if err != nil {
			return interstateAnswers{}, err
		}
	}

	// Item 8 - Changes every trip, so it's always asked
	answers.TransportingTo, err = internal_funcs.PromptRequiredText(reader, "Item 8 - Address the firearms are transported to")
	if err != nil {
		return interstateAnswers{}, err
	}
	answers.TransportingToCounty, err = internal_funcs.PromptRequiredText(reader, "Item 8 - County the firearms are transported to")
	if err != nil {
		return interstateAnswers{}, err
	}
	// Item 9 is a single line across the page, so a long carrier name and address is checked against its width
	if settings.Interstate.ModeOfTransportation != "" {
		answers.ModeOfTransportation = settings.Interstate.ModeOfTransportation
	} else {
		answers.ModeOfTransportation, err = internal_funcs.PromptValidatedText(reader,
			"Item 9 - Mode of transportation (the carrier's name and address if shipping)",
			fitsField(fieldWidths, interstatePage+"TextField16[0]", nil))
		if err != nil {
			return interstateAnswers{}, err
		}
	}
	return answers, nil
}

// promptInterstateFirearm Ask for the seven Item 4 columns for one firearm
func promptInterstateFirearm(reader *bufio.Reader, row int, firearmTypes []string, fieldWidths map[string]float64) (interstateFirearm, error) {
	var firearm interstateFirearm
	var err error
	fmt.Printf("\nItem 4 - Firearm %d of up to %d\n", row, interstateMaxFirearms)

	// columnField The full field name of this row's cell in a column
	columnField := func(column string) string {
		return interstatePage + fmt.Sprintf(column, row)
	}
	columns := interstateFirearmColumns

	firearm.Manufacturer, err = internal_funcs.PromptValidatedText(reader, "Item 4a - Manufacturer",
		fitsField(fieldWidths, columnField(columns.Manufacturer), nil))
	if err != nil {
		return interstateFirearm{}, err
	}
	typeChoice, err := internal_funcs.PromptChoice(reader, "Item 4b - Type of firearm:", firearmTypes)
	if err != nil {
		return interstateFirearm{}, err
	}
	firearm.FirearmType = firearmTypes[typeChoice]

	// The rest are free text, asked in the order of the form's columns
	textColumns := []struct {
		question string
		column   string
		format   func(answer string) error
		answer   *string
	}{
		{"Item 4c - Caliber or gauge (a number, like 556 or 300)", columns.Caliber, validateCaliber, &firearm.Caliber},
		{"Item 4d - Model", columns.Model, nil, &firearm.Model},
		{"Item 4e - Barrel length in inches (like 10 or 10.5)", columns.BarrelLength, validateLength, &firearm.BarrelLength},
		{"Item 4f - Overall length in inches (like 26 or 26.5)", columns.OverallLength, validateLength, &firearm.OverallLength},
		{"Item 4g - Serial number", columns.SerialNumber, nil, &firearm.SerialNumber},
	}
	for _, textColumn := range textColumns {
		*textColumn.answer, err = internal_funcs.PromptValidatedText(reader, textColumn.question,
			fitsField(fieldWidths, columnField(textColumn.column), textColumn.format))
		if err != nil {
			return interstateFirearm{}, err
		}
	}
	return firearm, nil
}

// promptFormDate Ask for a MM/DD/YYYY date, rejecting one before notBefore, which the error calls notBeforeName
func promptFormDate(reader *bufio.Reader, question string, notBefore time.Time, notBeforeName string) (time.Time, error) {
	var parsedDate time.Time
	_, err := internal_funcs.PromptValidatedText(reader, question, func(answer string) error {
		if !formDatePattern.MatchString(answer) {
			return errors.New("dates use the format MM/DD/YYYY with leading zeroes")
		}
		date, err := time.Parse(formDateLayout, answer)
		if err != nil {
			return errors.New("that isn't a real date")
		}
		if date.Before(notBefore) {
			return fmt.Errorf("this date can't be before %s (%s)", notBeforeName, notBefore.Format(formDateLayout))
		}
		parsedDate = date
		return nil
	})
	return parsedDate, err
}

// fitsField Build a validator that requires an answer, runs format on it if given, and rejects it if it won't fit
// in fieldName even at minimumFontSize
func fitsField(fieldWidths map[string]float64, fieldName string, format func(answer string) error) func(answer string) error {
	return func(answer string) error {
		if answer == "" {
			return errors.New("an answer is required")
		}
		if format != nil {
			if err := format(answer); err != nil {
				return err
			}
		}
		fits, err := textFitsWidth(answer, fieldWidths[fieldName], minimumFontSize)
		if err != nil {
			return err
		}
		if !fits {
			return errors.New("that's too long to fit on the form, so please shorten it")
		}
		return nil
	}
}

// validateCaliber Require a caliber or gauge written as a whole number, the way ATF forms expect (556, not 5.56 NATO)
func validateCaliber(caliber string) error {
	if !caliberPattern.MatchString(caliber) {
		return errors.New("calibers are whole numbers like 556 or 300, without a leading dot or a name")
	}
	return nil
}

// validateLength Require a length written as a number of inches with no unit, like 10 or 10.5
func validateLength(length string) error {
	if !lengthPattern.MatchString(length) {
		return errors.New("lengths are numbers of inches like 10 or 10.5, without \" or a unit")
	}
	return nil
}

// printInterstateSummary Show the user everything going on the 5320.20 so they can confirm it before anything is written
func printInterstateSummary(settings *internal_funcs.FFConfig, answers interstateAnswers) {
	ownerName, ownerType := interstateOwner(settings)
	fmt.Println("\n--- 5320.20 Review ---")
	fmt.Printf("Item 1  Owner:      %s (%s)\n", ownerName, ownerType)
	fmt.Printf("Registered in NFRTR: %s\n", yesOrNo(answers.RegisteredInNFRTR))
	fmt.Printf("Item 2  Returning:  %s\n", yesOrNo(answers.ReturnToOriginalLocation))
	fmt.Printf("Item 3  Dates:      %s to %s\n", answers.StartDate.Format(formDateLayout), answers.EndDate.Format(formDateLayout))
	for index, firearm := range answers.Firearms {
		fmt.Printf("Item 4  Firearm %d:  %s %s, %s, caliber %s, %s\" barrel, %s\" overall, serial %s\n", index+1,
			firearm.Manufacturer, firearm.Model, firearm.FirearmType, firearm.Caliber,
			firearm.BarrelLength, firearm.OverallLength, firearm.SerialNumber)
	}
	if answers.ReturnedFormMailingAddress == "" {
		fmt.Println("Item 5  Return to:  The Item 1 address")
	} else {
		fmt.Printf("Item 5  Return to:  %s\n", answers.ReturnedFormMailingAddress)
	}
	fmt.Printf("Item 6  Reason:     %s\n", answers.Reason)
	fmt.Printf("Item 7  From:       %s (%s)\n", answers.TransportingFrom, countyName(answers.TransportingFromCounty))
	fmt.Printf("Item 8  To:         %s (%s)\n", answers.TransportingTo, countyName(answers.TransportingToCounty))
	fmt.Printf("Item 9  Transport:  %s\n", answers.ModeOfTransportation)
}

// buildInterstateFields Map the settings and answers onto the 5320.20's field names
func buildInterstateFields(settings *internal_funcs.FFConfig, answers interstateAnswers) formFields {
	fields := formFields{TextFields: map[string]string{}}
	page := interstatePage

	// Item 1 - The registered owner is the trust when there is one
	if settings.UserData.TrustName != "" {
		fields.TextFields[page+"TextField2[0]"] = settings.UserData.TrustName + "\n" + settings.UserData.TrustAddress
		fields.CheckedBoxes = append(fields.CheckedBoxes, page+"tru[0]")
	} else {
		fields.TextFields[page+"TextField2[0]"] = settings.UserData.FullLegalName + "\n" + settings.UserData.FullLegalAddress
		fields.CheckedBoxes = append(fields.CheckedBoxes, page+"ind[0]")
	}
	fields.TextFields[page+"TextField3[0]"] = settings.UserData.PhoneNumber
	fields.TextFields[page+"TextField4[0]"] = settings.UserData.EmailAddress

	// Item 2 - Whether the firearms come back
	if answers.ReturnToOriginalLocation {
		fields.CheckedBoxes = append(fields.CheckedBoxes, page+"yes[0]")
	} else {
		fields.CheckedBoxes = append(fields.CheckedBoxes, page+"no[0]")
	}

	// Item 3 - The dates. The form's "From" date field shares a name with the e-mail field, so it's TextField4[1]
	fields.TextFields[page+"TextField4[1]"] = answers.StartDate.Format(formDateLayout)
	fields.TextFields[page+"TextField4to[0]"] = answers.EndDate.Format(formDateLayout)

	// Item 4 - One row per firearm
	columns := interstateFirearmColumns
	for index, firearm := range answers.Firearms {
		row := index + 1
		fields.TextFields[page+fmt.Sprintf(columns.Manufacturer, row)] = firearm.Manufacturer
		fields.TextFields[page+fmt.Sprintf(columns.FirearmType, row)] = firearm.FirearmType
		fields.TextFields[page+fmt.Sprintf(columns.Caliber, row)] = firearm.Caliber
		fields.TextFields[page+fmt.Sprintf(columns.Model, row)] = firearm.Model
		fields.TextFields[page+fmt.Sprintf(columns.BarrelLength, row)] = firearm.BarrelLength
		fields.TextFields[page+fmt.Sprintf(columns.OverallLength, row)] = firearm.OverallLength
		fields.TextFields[page+fmt.Sprintf(columns.SerialNumber, row)] = firearm.SerialNumber
	}

	// Item 5 - Left blank unless the approved form goes somewhere other than the Item 1 address
	if answers.ReturnedFormMailingAddress != "" {
		fields.TextFields[page+"TextField124[0]"] = answers.ReturnedFormMailingAddress
	}

	// Items 6 through 9 - The trip. The county gets its own line under each address.
	fields.TextFields[page+"TextField13[0]"] = answers.Reason
	fields.TextFields[page+"TextField15[0]"] = answers.TransportingFrom + "\n" + countyName(answers.TransportingFromCounty)
	fields.TextFields[page+"TextField15transport[0]"] = answers.TransportingTo + "\n" + countyName(answers.TransportingToCounty)
	fields.TextFields[page+"TextField16[0]"] = answers.ModeOfTransportation
	return fields
}

// interstateFilePath Build the output path for a 5320.20: {TravelStartDate}_{Owner}_{ItemCount}Gun(s)_5320_20.pdf
func interstateFilePath(settings *internal_funcs.FFConfig, answers interstateAnswers) string {
	owner := fileNamePart(settings.UserData.TrustName)
	if owner == "" {
		owner = fileNamePart(lastName(settings.UserData.FullLegalName))
	}
	gunWord := "Guns"
	if len(answers.Firearms) == 1 {
		gunWord = "Gun"
	}
	fileName := fmt.Sprintf("%s_%s_%d%s_5320_20.pdf", answers.StartDate.Format(fileNameDateLayout), owner, len(answers.Firearms), gunWord)
	return filepath.Join(settings.Settings.OutputPath, fileName)
}

// interstateOwner Return the registered owner's name and type for Item 1: the trust when there is one, otherwise the individual
func interstateOwner(settings *internal_funcs.FFConfig) (string, string) {
	if settings.UserData.TrustName != "" {
		return settings.UserData.TrustName, "Trust"
	}
	return settings.UserData.FullLegalName, "Individual"
}

// countyName Add " County" after a county's name unless it already ends with County, Parish or Borough
func countyName(county string) string {
	countyWords := strings.Fields(strings.ToLower(county))
	if len(countyWords) > 0 && slices.Contains(countyNameEndings, countyWords[len(countyWords)-1]) {
		return county
	}
	return county + " County"
}

// yesOrNo Spell out a yes or no answer for the review
func yesOrNo(answer bool) string {
	if answer {
		return "Yes"
	}
	return "No"
}
