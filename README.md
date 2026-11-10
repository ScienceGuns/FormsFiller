Mad Scientist Research Forms Filler
===================================
This tool fills out ATF forms for [NFA eForms](https://eforms.atf.gov/) submissions. The details that stay the same from
form to form live in a settings file, and Forms Filler prompts for the details unique to each form before saving PDFs
that are ready to sign and submit.

Forms Filler is built and used by [Mad Scientist Research LLC](https://scienceguns.com/), and anyone else is welcome to
use it under the MIT license.

**Forms Filler is not legal advice.** You are responsible for everything on a form you submit to ATF. Review every form
before you sign it.

Supported Forms
---------------

| Form    | Name                              | Settings it needs                |
|---------|-----------------------------------|----------------------------------|
| 5320.23 | Responsible Persons Questionnaire | `trust-name` and `trust-address` |

A form your settings don't support is left off the menu.

Installing
----------
Forms Filler is built for `darwin-arm64` and `linux-amd64`. On macOS, install it with Homebrew:

```shell
brew install scienceguns/taps/formsfiller
```

Settings File
-------------
The settings file belongs in `~/.config/scienceguns/formsfiller.json`. A different file can be used with
`--settings-path`.

The settings file holds your SSN and date of birth, so **it must be mode `0400`** (readable only by you). Forms Filler
refuses to run until it is:

```shell
chmod 0400 ~/.config/scienceguns/formsfiller.json
```

Currently supported values:

```json
{
   "settings": {
      "output-path": "~/Desktop"
   },
   "user-data": {
      "trust-name": "The Krieger Clone Trust",
      "trust-address": "1 Figgis Agency Plaza, New York, NY 10001",
      "full-legal-name": "Algernop Krieger",
      "full-legal-address": "1 Figgis Agency Plaza, New York, NY 10001",
      "phone-number": "212-555-0187",
      "email-address": "dr.krieger@figgis.example",
      "other-names-used": "Dr. Krieger",
      "social-security-number": "078-05-1120",
      "date-of-birth": "09/24/1977",
      "ethnicity": "Not Hispanic or Latino",
      "race": ["White"],
      "country-of-citizenship": "",
      "state-of-birth": "New York",
      "country-of-birth": "United States of America"
   },
   "cleo": {
      "agency-name": "New York City Police Department",
      "official-name-and-title": "Barry Dylan, Chief of Police",
      "mailing-address": "1 Police Plaza, New York, NY 10007"
   }
}
```

Every key in the example is required except these optional ones:

1. `settings.output-path` (Default: `~/Desktop`) The folder finished forms are saved in. A leading `~` works.
2. `user-data.trust-name` (Default: blank) The trust or legal entity you are a responsible person for. Leaving it blank
   turns off the 5320.23.
3. `user-data.trust-address` (Default: blank) The trust or legal entity's address. Required when `trust-name` is set.
4. `user-data.other-names-used` (Default: blank) Aliases or a maiden name.
5. `user-data.country-of-citizenship` (Default: blank, meaning the U.S.A.) Any other country checks the "Other" box in
   Item 9a instead of the U.S.A. box.
6. `user-data.state-of-birth` (Default: blank) Required when `country-of-birth` is the U.S.

Some values have to be in a specific format:

1. `phone-number` - `123-456-7890`
2. `social-security-number` - `123-45-6789`. Numbers the SSA never issues are rejected.
3. `date-of-birth` - `MM/DD/YYYY` with leading zeroes, such as `09/24/1977`
4. `ethnicity` - `Hispanic or Latino` or `Not Hispanic or Latino`
5. `race` - A list of one or more of `American Indian or Alaska Native`, `Asian`, `Black or African American`,
   `Native Hawaiian or Other Pacific Islander` and `White`

A misspelled key is an error rather than being ignored, so a typo can't quietly leave a field blank on a form.

Filling Out A Form
------------------
Run `formsfiller` with no arguments:

1. Your settings file is checked. You only see this step if there's a problem.
2. Pick the form to fill out.
3. Answer the questions for the details unique to this form.
4. Review your answers. Answer `n` to start the questions over.
5. The finished form is saved to your `output-path`.

Form 5320.23
------------

The 5320.23 asks which ATF form it's being submitted with (Item 1) and about the firearm (Item 4). The maker's address
can be left blank, since approved forms don't always list one.

Each run saves two files, named `{Serial}_{Trust}_{LastName}_RPQ_ATF.pdf` and `{Serial}_{Trust}_{LastName}_RPQ_CLEO.pdf`
with spaces removed:

1. The ATF copy (pages 1-4) to upload with your eForms application.
2. The CLEO copy (pages 5-6) to send to your chief law enforcement officer. It has no SSN or serial number, matching
   ATF's own CLEO copy.

Forms Filler assumes the responsible person can pass a NICS check, so it always answers:

| Item           | Answer                                                 |
|----------------|--------------------------------------------------------|
| 6.a - 6.m.1    | No                                                     |
| 6.m.2          | N/A, since it only applies after a Yes on 6.m.1        |
| 7              | Left blank (alien or admission number)                 |
| 8              | No UPIN ([UPIN support coming soon](TODO.md))          |
| 9a             | `U.S.A.`, unless `country-of-citizenship` is set       |
| Signature/Date | Left blank. Sign and date both copies yourself.        |
| 3e (Photo)     | Left blank. eForms takes your photo as its own upload. |

If any of these answers aren't true for you, fill out your 5320.23 by hand instead.

Privacy
-------
Forms Filler doesn't connect to anything, so it works offline.

1. Your settings file must be mode `0400`, and saved forms are mode `0600`.
2. Error messages never include your SSN.
3. The CLEO copy never contains your SSN or the firearm's serial number.
4. The first run creates `~/.config/scienceguns/pdfcpu/` with pdfcpu's settings and the font it fills forms with. The
   font is built into Forms Filler, so nothing is downloaded.

The blank ATF form is built into Forms Filler. The pages themselves are never changed. Each saved copy is the official
form with your answers filled in, unlocked.

Dependencies
------------

1. [pdfcpu](https://github.com/pdfcpu/pdfcpu) (Apache-2.0) fills out the PDF forms
2. [Cobra](https://github.com/spf13/cobra) (Apache-2.0) runs the command line interface
