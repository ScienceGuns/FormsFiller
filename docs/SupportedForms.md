Supported Forms
===============
The ATF forms Forms Filler can fill out, in the order they appear on its menu. A form your settings don't support is
left off the menu. The settings are described in the [README](../README.md#settings-file).

| Form    | Name                                | Settings it needs                |
|---------|-------------------------------------|----------------------------------|
| 5320.23 | Responsible Persons Questionnaire   | `trust-name` and `trust-address` |
| 5320.20 | Application to Transport Interstate | Nothing extra                    |

Form 5320.23
------------

The 5320.23 asks which ATF form it's being submitted with (Item 1) and about the firearm (Item 4). The maker's address
can be left blank, since approved forms don't always list one. Caliber or gauge is a whole number, like `556` or `300`,
the same as on the 5320.20.

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
| 8              | No UPIN ([UPIN support coming soon](../TODO.md))       |
| 9a             | `U.S.A.`, unless `country-of-citizenship` is set       |
| Signature/Date | Left blank. Sign and date both copies yourself.        |
| 3e (Photo)     | Left blank. eForms takes your photo as its own upload. |

If any of these answers aren't true for you, fill out your 5320.23 by hand instead.

Form 5320.20
------------

The 5320.20 is the application to transport short-barreled rifles, short-barreled shotguns, machineguns and destructive
devices across state lines. The registered owner in Item 1 is your trust when `trust-name` is set, and you otherwise.

Forms Filler first asks whether the firearms are registered to the owner in the NFRTR. ATF won't take registered and
unregistered firearms on the same form, and only short-barreled rifles and shotguns can go on one for unregistered
firearms. The answer also decides where the form goes:

| Firearms                    | Submit through                       |
|-----------------------------|--------------------------------------|
| Registered in the NFRTR     | eForms, or email to `nfafax@atf.gov` |
| Not registered in the NFRTR | Email to `nfafax@atf.gov` only       |

Items 2, 5, 6, 7 and 9 come from the `interstate` settings when they're set and are asked for when they aren't. The
dates (Item 3), the firearms (Item 4) and the destination (Item 8) are asked for every time. The start date can't be
before today, and the end date can't be before the start date. For each firearm:

1. Caliber or gauge is a whole number, like `556` or `300`
2. Barrel and overall length are numbers of inches, like `10` or `10.5`

Item 4 holds up to three firearms, and more need another 5320.20. Some of its columns are narrow, so a long answer is
printed in a smaller font to fit, and one that won't fit even then is asked for again.

The form is saved as one PDF with both of ATF's pages, named `{TravelStartDate}_{Owner}_{ItemCount}Gun(s)_5320_20.pdf`,
such as `2009-09-17_TheKriegerCloneTrust_1Gun_5320_20.pdf`. `{Owner}` is the trust name, or your last name without a
trust. Items 10-12 (temporary export) and the signature and date are left blank.
