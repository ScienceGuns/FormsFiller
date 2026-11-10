/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package cmd

import (
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"runtime"
)

// The version string should be updated before any merge to main
var shortVersion = "0.1.0"
var projectMaintainer = "Mad Scientist Research LLC"
var projectLicense = "MIT"
var functionHelpShort = "Fill out ATF forms for NFA eForms submissions"
var functionHelpLong = `Forms Filler fills out ATF forms for NFA eForms submissions using the reusable
details in a settings file, then prompts for the details unique to each form.`
var buildArch = runtime.GOARCH
var SettingsFilePath, _ = internal_funcs.OutputUserPath(".config/scienceguns/formsfiller.json")
