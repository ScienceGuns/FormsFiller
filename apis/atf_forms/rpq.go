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
	"strings"
)

// checkRPQSettings Check that the settings have what Form 5320.23 needs, which is a trust or legal entity to be a responsible person for
func checkRPQSettings(settings *internal_funcs.FFConfig) error {
	if settings.UserData.TrustName == "" {
		return errors.New("trust-name is blank")
	}
	return nil
}

// fillRPQ Guide the user through the details unique to a 5320.23, then save its ATF and CLEO copies
func fillRPQ(reader *bufio.Reader, settings *internal_funcs.FFConfig) error {
	// Keep asking until the user confirms the answers look right
	var answers rpqAnswers
	for {
		var err error
		answers, err = promptRPQAnswers(reader)
		if err != nil {
			return err
		}
		printRPQSummary(settings, answers)
		confirmed, err := internal_funcs.PromptYesNo(reader, "\nDoes this look correct?", true)
		if err != nil {
			return err
		}
		if confirmed {
			break
		}
		fmt.Println("\nLet's try that again.")
	}

	// Check for earlier copies of this form before doing any work
	for _, formCopy := range rpqCopies {
		outputPath := rpqFilePath(settings, answers, formCopy.Name)
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
	}

	// Item 3g has to be drawn as a single line on both copies or the date of birth gets cut off
	var dateOfBirthFields []string
	for _, formCopy := range rpqCopies {
		dateOfBirthFields = append(dateOfBirthFields, formCopy.DateOfBirthField)
	}
	rpqTemplateFixed, err := clearMultilineFlag(rpqTemplate, dateOfBirthFields)
	if err != nil {
		return err
	}

	// Fill both copies at once, then split them into their own files
	filledPDF, err := fillPDF(rpqTemplateFixed, buildRPQFields(settings, answers))
	if err != nil {
		return err
	}
	fmt.Println()
	for _, formCopy := range rpqCopies {
		copyPDF, err := extractPages(filledPDF, formCopy.Pages)
		if err != nil {
			return err
		}
		outputPath := rpqFilePath(settings, answers, formCopy.Name)
		if err := writePDF(outputPath, copyPDF); err != nil {
			return err
		}
		fmt.Printf("Saved the %s copy to %s\n", formCopy.Name, outputPath)
	}

	// Remind the user of everything Forms Filler leaves to them. Item 3e (photo) stays blank since eForms takes
	// the photo and EFT fingerprints as their own uploads
	fmt.Println("\nBefore submitting:")
	fmt.Println("  1. Sign and date both copies.")
	fmt.Println("  2. Upload the ATF copy with your eForms application, along with your photo and EFT file.")
	fmt.Println("  3. Send the CLEO copy to your CLEO.")
	return nil
}

// rpqFilePath Build the output path for one copy of a 5320.23: {Serial}_{Trust}_{LastName}_RPQ_{Copy}.pdf
func rpqFilePath(settings *internal_funcs.FFConfig, answers rpqAnswers, copyName string) string {
	fileName := fmt.Sprintf("%s_%s_%s_RPQ_%s.pdf",
		fileNamePart(answers.SerialNumber),
		fileNamePart(settings.UserData.TrustName),
		fileNamePart(lastName(settings.UserData.FullLegalName)),
		copyName)
	return filepath.Join(settings.Settings.OutputPath, fileName)
}

// promptRPQAnswers Ask for the 5320.23 details that change with every firearm
func promptRPQAnswers(reader *bufio.Reader) (rpqAnswers, error) {
	var answers rpqAnswers

	// Item 1 - The ATF form this questionnaire goes with
	atfFormLabels := make([]string, len(rpqATFFormOptions))
	for index, atfFormOption := range rpqATFFormOptions {
		atfFormLabels[index] = atfFormOption.Label
	}
	atfFormChoice, err := internal_funcs.PromptChoice(reader, "\nItem 1 - Which ATF form is this questionnaire being submitted with?", atfFormLabels)
	if err != nil {
		return rpqAnswers{}, err
	}
	answers.ATFForm = rpqATFFormOptions[atfFormChoice]

	// Item 4a - The type of NFA firearm
	firearmTypeChoice, err := internal_funcs.PromptChoice(reader, "\nItem 4a - Type of firearm:", rpqFirearmTypes)
	if err != nil {
		return rpqAnswers{}, err
	}
	answers.FirearmType = rpqFirearmTypes[firearmTypeChoice]

	// Items 4b through 4e are free text, asked in the order they appear on the form
	fmt.Println()
	// The maker's address can be blank since approved forms don't always list one (e.g. B&T USA LLC)
	textPrompts := []struct {
		question   string
		answer     *string
		allowBlank bool
	}{
		{"Item 4b - Name of the maker, manufacturer and/or importer", &answers.MakerName, false},
		{"Item 4b - Address of the maker, manufacturer and/or importer", &answers.MakerAddress, true},
		{"Item 4c - Firearm model", &answers.Model, false},
		{"Item 4d - Caliber or gauge", &answers.Caliber, false},
		{"Item 4e - Firearm serial number", &answers.SerialNumber, false},
	}
	for _, textPrompt := range textPrompts {
		if textPrompt.allowBlank {
			*textPrompt.answer, err = internal_funcs.PromptOptionalText(reader, textPrompt.question)
		} else {
			*textPrompt.answer, err = internal_funcs.PromptRequiredText(reader, textPrompt.question)
		}
		if err != nil {
			return rpqAnswers{}, err
		}
	}
	return answers, nil
}

// printRPQSummary Show the user who and what the 5320.23 is for so they can confirm it before anything is written
func printRPQSummary(settings *internal_funcs.FFConfig, answers rpqAnswers) {
	fmt.Println("\n--- 5320.23 Review ---")
	fmt.Printf("Trust:              %s\n", settings.UserData.TrustName)
	fmt.Printf("Responsible Person: %s\n", settings.UserData.FullLegalName)
	fmt.Printf("Item 1  ATF form:   %s\n", answers.ATFForm.Label)
	fmt.Printf("Item 4a Type:       %s\n", answers.FirearmType)
	if answers.MakerAddress == "" {
		fmt.Printf("Item 4b Maker:      %s (no address)\n", answers.MakerName)
	} else {
		fmt.Printf("Item 4b Maker:      %s, %s\n", answers.MakerName, answers.MakerAddress)
	}
	fmt.Printf("Item 4c Model:      %s\n", answers.Model)
	fmt.Printf("Item 4d Caliber:    %s\n", answers.Caliber)
	fmt.Printf("Item 4e Serial:     %s\n", answers.SerialNumber)
}

// buildRPQFields Map the settings and answers onto the field names of both copies of the 5320.23
func buildRPQFields(settings *internal_funcs.FFConfig, answers rpqAnswers) formFields {
	fields := formFields{TextFields: map[string]string{}}
	userData := settings.UserData

	for _, formCopy := range rpqCopies {
		front := formCopy.FrontPage
		back := formCopy.BackPage

		// Item 1 - The ATF form this questionnaire goes with
		fields.CheckedBoxes = append(fields.CheckedBoxes, front+answers.ATFForm.FieldName)

		// Item 2 - The trust or legal entity
		fields.TextFields[front+"applicantaddress[0]"] = userData.TrustName + "\n" + userData.TrustAddress

		// Item 3 - The responsible person
		fields.TextFields[front+"responsibleaddress[0]"] = userData.FullLegalName + "\n" + userData.FullLegalAddress
		fields.TextFields[front+"telephone[0]"] = userData.PhoneNumber
		fields.TextFields[front+"email[0]"] = userData.EmailAddress
		fields.TextFields[front+"othernames[0]"] = userData.OtherNamesUsed
		fields.TextFields[front+formCopy.DateOfBirthField] = userData.DateOfBirth
		fields.CheckedBoxes = append(fields.CheckedBoxes, front+rpqEthnicityBoxes[userData.Ethnicity])
		for _, race := range userData.Race {
			fields.CheckedBoxes = append(fields.CheckedBoxes, front+rpqRaceBoxes[race])
		}

		// Item 4 - The firearm
		fields.TextFields[front+"firearmtype[0]"] = answers.FirearmType
		fields.TextFields[front+"importeraddress[0]"] = strings.TrimSuffix(answers.MakerName+"\n"+answers.MakerAddress, "\n")
		fields.TextFields[front+"Model[0]"] = answers.Model
		fields.TextFields[front+"caliber[0]"] = answers.Caliber

		// The SSN and serial number only exist on the ATF copy
		if formCopy.HasSSNAndSerial {
			fields.TextFields[front+"ssn2f[0]"] = userData.SocialSecurityNumber
			fields.TextFields[front+"serial[0]"] = answers.SerialNumber
		}

		// Item 5 - The CLEO being notified
		fields.TextFields[front+"TextField3[0]"] = settings.CLEO.AgencyName
		fields.TextFields[front+"TextField4[0]"] = settings.CLEO.OfficialNameAndTitle
		fields.TextFields[front+"TextField5[0]"] = settings.CLEO.MailingAddress

		// Items 6 and 8 - The answers that never change
		for _, backPageBox := range rpqBackPageBoxes {
			fields.CheckedBoxes = append(fields.CheckedBoxes, back+backPageBox)
		}

		// Item 9a - A blank country of citizenship means U.S.A., anything else replaces it
		if userData.CountryOfCitizenship == "" {
			fields.CheckedBoxes = append(fields.CheckedBoxes, back+"usacheckbox[0]")
		} else {
			fields.CheckedBoxes = append(fields.CheckedBoxes, back+"othercountrycheckbox[0]")
			fields.TextFields[back+"Othercountry[0]"] = userData.CountryOfCitizenship
		}

		// Items 9b and 9c - Where the responsible person was born
		fields.TextFields[back+"statebirth[0]"] = userData.StateOfBirth
		fields.TextFields[back+"statecountry[0]"] = userData.CountryOfBirth
	}
	return fields
}
