Testing
=======
Run the unit tests with:

```shell
go test ./...
```

The tests never touch your real settings file or Desktop. Every file they write goes in a temporary directory.

Testing The Whole Flow
----------------------
To run Forms Filler by hand without your own details, save this as `krieger.json` somewhere outside the repo and point
Forms Filler at it with `--settings-path`. Set `output-path` to a folder that exists.

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

```shell
chmod 0400 krieger.json
go run . --settings-path krieger.json
```

The sample firearm used throughout the tests is The Gauss, a short-barreled rifle made by Mad Scientist Research LLC
in Perkiomenville, PA 18074, in caliber 300 with serial number `TGB126MS0001`.

Note
----
Currently, Forms Filler is built for `darwin-arm64` and `linux-amd64`, however interactive testing is only performed on
the macOS variant.
