/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package atf_forms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// pdfConfiguration Load the pdfcpu configuration from pdfcpuConfigRoot, creating it the first time
func pdfConfiguration() (*model.Configuration, error) {
	// pdfcpu needs its config directory for the Roboto font it fills form fields with, so stateless mode won't work
	pdfConfig, err := api.LoadConfiguration(api.ConfigurationOptions{Root: pdfcpuConfigRoot})
	if err != nil {
		return nil, fmt.Errorf("failed to load the pdfcpu configuration from %s: %w", pdfcpuConfigRoot, err)
	}
	return pdfConfig, nil
}

// readFormContext Read a PDF into a pdfcpu context so its form fields can be changed
func readFormContext(pdfData []byte) (*model.Context, error) {
	pdfConfig, err := pdfConfiguration()
	if err != nil {
		return nil, err
	}
	pdfContext, err := api.ReadContext(context.Background(), bytes.NewReader(pdfData), pdfConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read the PDF template: %w", err)
	}
	return pdfContext, nil
}

// writeFormContext Write a pdfcpu context back out as PDF data
func writeFormContext(pdfContext *model.Context) ([]byte, error) {
	var updatedPDF bytes.Buffer
	if err := api.WriteContext(context.Background(), pdfContext, &updatedPDF); err != nil {
		return nil, fmt.Errorf("failed to write the prepared PDF template: %w", err)
	}
	return updatedPDF.Bytes(), nil
}

// visitFormFields Call visit with the full name and dictionary of every field in a PDF's form
func visitFormFields(xRefTable *model.XRefTable, visit func(fullName string, fieldDict types.Dict) error) error {
	// pdfcpu loads objects as they're asked for, so walk the form's field tree down from the AcroForm dictionary
	catalogDict, err := xRefTable.Catalog()
	if err != nil {
		return fmt.Errorf("failed to read the PDF catalog: %w", err)
	}
	acroFormDict, err := xRefTable.DereferenceDict(catalogDict["AcroForm"])
	if err != nil || acroFormDict == nil {
		return fmt.Errorf("failed to find the PDF's form: %v", err)
	}
	topFields, err := xRefTable.DereferenceArray(acroFormDict["Fields"])
	if err != nil {
		return fmt.Errorf("failed to read the PDF's form fields: %w", err)
	}

	// Each field's full name is its parents' names and its own, joined with dots
	type fieldToVisit struct {
		object     types.Object
		parentName string
	}
	var fieldsToVisit []fieldToVisit
	for _, topField := range topFields {
		fieldsToVisit = append(fieldsToVisit, fieldToVisit{object: topField})
	}
	for len(fieldsToVisit) > 0 {
		current := fieldsToVisit[0]
		fieldsToVisit = fieldsToVisit[1:]
		fieldDict, err := xRefTable.DereferenceDict(current.object)
		if err != nil || fieldDict == nil {
			return fmt.Errorf("failed to read a PDF form field: %v", err)
		}

		// A widget with no name of its own belongs to its parent field
		fullName := current.parentName
		if partialName, err := xRefTable.DereferenceStringOrHexLiteral(fieldDict["T"], model.V10, nil); err == nil && partialName != "" {
			fullName = partialName
			if current.parentName != "" {
				fullName = current.parentName + "." + partialName
			}
		}
		if err := visit(fullName, fieldDict); err != nil {
			return err
		}

		// Fields nested under this one get visited too
		if kids, err := xRefTable.DereferenceArray(fieldDict["Kids"]); err == nil {
			for _, kid := range kids {
				fieldsToVisit = append(fieldsToVisit, fieldToVisit{object: kid, parentName: fullName})
			}
		}
	}
	return nil
}

// clearMultilineFlag Return a copy of a PDF with the multi-line flag cleared on the text fields with these full names.
// Item 3g (date of birth) on the 5320.23 is marked multi-line in a box too short for one line of its 9pt font,
// and multi-line text starts at the top of the box, so the date gets cut off unless it's drawn as a single line.
func clearMultilineFlag(pdfData []byte, fieldNames []string) ([]byte, error) {
	pdfContext, err := readFormContext(pdfData)
	if err != nil {
		return nil, err
	}
	xRefTable := pdfContext.XRefTable

	// Clear the flag on each named field
	clearedFields := 0
	err = visitFormFields(xRefTable, func(fullName string, fieldDict types.Dict) error {
		if !slices.Contains(fieldNames, fullName) {
			return nil
		}
		fieldFlags := 0
		if flagsEntry, err := xRefTable.DereferenceInteger(fieldDict["Ff"]); err == nil && flagsEntry != nil {
			fieldFlags = flagsEntry.Value()
		}
		fieldDict["Ff"] = types.Integer(fieldFlags &^ multilineFieldFlag)
		clearedFields++
		return nil
	})
	if err != nil {
		return nil, err
	}
	// A count that doesn't match means the template changed and the field names need checking
	if clearedFields != len(fieldNames) {
		return nil, fmt.Errorf("failed to prepare the PDF template: expected %d fields to make single line, but found %d", len(fieldNames), clearedFields)
	}
	return writeFormContext(pdfContext)
}

// readFieldWidths Return the width in points of every form field in a PDF that has a box on the page
func readFieldWidths(pdfData []byte) (map[string]float64, error) {
	pdfContext, err := readFormContext(pdfData)
	if err != nil {
		return nil, err
	}
	xRefTable := pdfContext.XRefTable

	fieldWidths := map[string]float64{}
	err = visitFormFields(xRefTable, func(fullName string, fieldDict types.Dict) error {
		fieldRect, err := xRefTable.DereferenceArray(fieldDict["Rect"])
		if err != nil || len(fieldRect) != 4 {
			return nil // This is not an error since a field without its own box is just a parent of other fields
		}
		rectangle, err := xRefTable.RectForArray(fieldRect)
		if err != nil {
			return fmt.Errorf("failed to read the box of form field %s: %w", fullName, err)
		}
		fieldWidths[fullName] = rectangle.Width()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fieldWidths, nil
}

// fitFontSizes Return a copy of a PDF where each single-line text field being filled has its font size lowered,
// one point at a time down to minimumFontSize, until its value fits. A value that still won't fit is an error.
func fitFontSizes(pdfData []byte, textValues map[string]string) ([]byte, error) {
	pdfContext, err := readFormContext(pdfData)
	if err != nil {
		return nil, err
	}
	xRefTable := pdfContext.XRefTable

	err = visitFormFields(xRefTable, func(fullName string, fieldDict types.Dict) error {
		value := textValues[fullName]
		if value == "" {
			return nil
		}
		// Multi-line fields wrap onto more lines instead of running past the edge
		fieldFlags := 0
		if flagsEntry, err := xRefTable.DereferenceInteger(fieldDict["Ff"]); err == nil && flagsEntry != nil {
			fieldFlags = flagsEntry.Value()
		}
		if fieldFlags&multilineFieldFlag != 0 {
			return nil
		}

		// Find the field's width and the font size it starts at
		fieldRect, err := xRefTable.DereferenceArray(fieldDict["Rect"])
		if err != nil || len(fieldRect) != 4 {
			return nil // This is not an error since a field without its own box draws nothing
		}
		rectangle, err := xRefTable.RectForArray(fieldRect)
		if err != nil {
			return fmt.Errorf("failed to read the box of form field %s: %w", fullName, err)
		}
		defaultAppearance, err := xRefTable.DereferenceStringOrHexLiteral(fieldDict["DA"], model.V10, nil)
		if err != nil {
			return nil // This is not an error since a field with no font set uses pdfcpu's default
		}
		fontSettings := fontSizePattern.FindStringSubmatch(defaultAppearance)
		if fontSettings == nil {
			return nil // This is not an error since a field with no font size can't be resized
		}
		startingSize, err := strconv.ParseFloat(fontSettings[2], 64)
		if err != nil {
			return fmt.Errorf("failed to read the font size of form field %s: %w", fullName, err)
		}

		// Step the size down until the value fits
		for fontSize := int(startingSize); fontSize >= minimumFontSize; fontSize-- {
			fits, err := textFitsWidth(value, rectangle.Width(), fontSize)
			if err != nil {
				return err
			}
			if fits {
				if fontSize != int(startingSize) {
					fieldDict["DA"] = types.StringLiteral(fmt.Sprintf("%s %d Tf%s", fontSettings[1], fontSize, fontSettings[3]))
				}
				return nil
			}
		}
		// The value isn't included since some fields hold PII
		return fmt.Errorf("failed to fill the PDF form: the value for form field %s is too long to fit even at %dpt", fullName, minimumFontSize)
	})
	if err != nil {
		return nil, err
	}
	return writeFormContext(pdfContext)
}

// textFitsWidth Report whether text fits inside a field fieldWidth points wide at fontSize, leaving room for the field's padding
func textFitsWidth(text string, fieldWidth float64, fontSize int) (bool, error) {
	// pdfcpu's text width measurements only see fonts in its font directory, which it only sets up
	// for its default config location, so point it at the directory under pdfcpuConfigRoot
	pdfConfig, err := pdfConfiguration()
	if err != nil {
		return false, err
	}
	fontDirectory, available := pdfConfig.UserFontStore()
	if !available || fontDirectory == "" {
		return false, errors.New("failed to find pdfcpu's font directory")
	}
	if font.UserFontDir != fontDirectory {
		font.UserFontDir = fontDirectory
		if err := font.ReloadUserFonts(context.Background()); err != nil {
			return false, fmt.Errorf("failed to load pdfcpu's fonts from %s: %w", fontDirectory, err)
		}
	}

	textWidth, err := font.TextWidthFloat(context.Background(), text, fillFontName, float64(fontSize))
	if err != nil {
		return false, fmt.Errorf("failed to measure text in %s: %w", fillFontName, err)
	}
	return textWidth <= fieldWidth-2*fieldPadding, nil
}

// fillPDF Fill a PDF form template with field values, check every value landed, and return the filled PDF
func fillPDF(templateData []byte, fields formFields) ([]byte, error) {
	pdfConfig, err := pdfConfiguration()
	if err != nil {
		return nil, err
	}

	// Build pdfcpu's form JSON in a fixed order, matching every field by its full name
	var pdfForm form.Form
	for _, fieldName := range slices.Sorted(maps.Keys(fields.TextFields)) {
		pdfForm.TextFields = append(pdfForm.TextFields, &form.TextField{Name: fieldName, Value: fields.TextFields[fieldName]})
	}
	for _, fieldName := range fields.CheckedBoxes {
		pdfForm.CheckBoxes = append(pdfForm.CheckBoxes, &form.CheckBox{Name: fieldName, Value: true})
	}
	formJSON, err := json.Marshal(form.FormGroup{Forms: []form.Form{pdfForm}})
	if err != nil {
		return nil, fmt.Errorf("failed to build the form data: %w", err)
	}

	// Fill the form. This is local work with nothing to time out, so a background context is enough
	var filledPDF bytes.Buffer
	if err := api.FillForm(context.Background(), bytes.NewReader(templateData), bytes.NewReader(formJSON), &filledPDF, pdfConfig); err != nil {
		return nil, fmt.Errorf("failed to fill the PDF form: %w", err)
	}

	// pdfcpu silently skips field names it can't find, so read every value back before trusting the result
	if err := verifyPDFFields(filledPDF.Bytes(), fields); err != nil {
		return nil, err
	}
	return filledPDF.Bytes(), nil
}

// verifyPDFFields Check that a filled PDF holds every text value and checked box it was supposed to get
func verifyPDFFields(pdfData []byte, fields formFields) error {
	pdfConfig, err := pdfConfiguration()
	if err != nil {
		return err
	}
	exportedForms, err := api.ExportForm(context.Background(), bytes.NewReader(pdfData), "verify", pdfConfig)
	if err != nil {
		return fmt.Errorf("failed to read back the filled PDF form: %w", err)
	}

	// Collect what's actually in the PDF
	textValues := map[string]string{}
	checkedBoxes := map[string]bool{}
	for _, exportedForm := range exportedForms.Forms {
		for _, textField := range exportedForm.TextFields {
			textValues[textField.Name] = textField.Value
		}
		// pdfcpu reports a text field holding a date (like Item 3g) as a date field once it's filled
		for _, dateField := range exportedForm.DateFields {
			textValues[dateField.Name] = dateField.Value
		}
		for _, checkBox := range exportedForm.CheckBoxes {
			checkedBoxes[checkBox.Name] = checkBox.Value
		}
	}

	// Compare it with what was asked for. Field values aren't included in errors since some hold PII
	for fieldName, wantValue := range fields.TextFields {
		gotValue, exists := textValues[fieldName]
		if !exists {
			return fmt.Errorf("failed to fill the PDF form: the template has no text field named %s", fieldName)
		}
		if gotValue != wantValue {
			return fmt.Errorf("failed to fill the PDF form: text field %s does not hold the expected value", fieldName)
		}
	}
	for _, fieldName := range fields.CheckedBoxes {
		if !checkedBoxes[fieldName] {
			return fmt.Errorf("failed to fill the PDF form: checkbox %s is missing or not checked", fieldName)
		}
	}
	return nil
}

// extractPages Return a new PDF holding only the given pages (e.g. "1-4") of another PDF
func extractPages(pdfData []byte, pageRange string) ([]byte, error) {
	pdfConfig, err := pdfConfiguration()
	if err != nil {
		return nil, err
	}
	var extractedPDF bytes.Buffer
	if err := api.Collect(context.Background(), bytes.NewReader(pdfData), &extractedPDF, []string{pageRange}, pdfConfig); err != nil {
		return nil, fmt.Errorf("failed to extract pages %s: %w", pageRange, err)
	}
	return extractedPDF.Bytes(), nil
}

// writePDF Write a PDF that only the owner can read, via a .partial file so a failed write never leaves half a form
func writePDF(destinationPath string, pdfData []byte) error {
	partialPath := destinationPath + ".partial"
	if err := os.WriteFile(partialPath, pdfData, 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", partialPath, err)
	}
	if err := os.Rename(partialPath, destinationPath); err != nil {
		// Don't leave a copy of the form lying around under the wrong name
		if removeErr := os.Remove(partialPath); removeErr != nil {
			log.Printf("Warning: Failed to remove %s: %v", partialPath, removeErr)
		}
		return fmt.Errorf("failed to rename %s to %s: %w", partialPath, destinationPath, err)
	}
	return nil
}

// fileNamePart Strip the whitespace and path separators out of a value so it can be part of a file name
func fileNamePart(value string) string {
	return strings.NewReplacer("/", "", "\\", "").Replace(strings.Join(strings.Fields(value), ""))
}

// lastName Return the last word of a full name, skipping suffixes like Jr. and III
func lastName(fullName string) string {
	nameParts := strings.Fields(strings.ReplaceAll(fullName, ",", " "))
	for index := len(nameParts) - 1; index >= 0; index-- {
		if !slices.Contains(nameSuffixes, strings.ToLower(strings.TrimSuffix(nameParts[index], "."))) {
			return nameParts[index]
		}
	}
	return "" // A name made of nothing but suffixes leaves the last name out of the file name
}
