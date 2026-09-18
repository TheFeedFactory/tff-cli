package normalize

import (
	"strings"
	"testing"
)

func findingCodes(fs []Finding) string {
	var codes []string
	for _, f := range fs {
		codes = append(codes, f.Code)
	}
	return strings.Join(codes, ",")
}

func TestReportsMissingPartsOfTheAddress(t *testing.T) {
	got := Inspect(Record{
		Location: Address{City: "Ede"},
	})

	if !strings.Contains(findingCodes(got), FindingStreetMissing) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingStreetMissing)
	}
	if !strings.Contains(findingCodes(got), FindingHouseNrMissing) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingHouseNrMissing)
	}
	if !strings.Contains(findingCodes(got), FindingZipcodeMissing) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingZipcodeMissing)
	}
}

func TestReportsAZipcodeItCannotFormat(t *testing.T) {
	got := Inspect(Record{Location: Address{Street: "Oude Arnhemseweg", HouseNr: "285", ZipCode: "7361", City: "Beekbergen"}})

	if !strings.Contains(findingCodes(got), FindingZipcodeUnrecognised) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingZipcodeUnrecognised)
	}
}

func TestReportsCoordinatesThatAreMissingOrNotInTheNetherlands(t *testing.T) {
	missing := Inspect(Record{Location: Address{Street: "Stationsweg", HouseNr: "6", ZipCode: "3771 VH", City: "Barneveld"}})
	if !strings.Contains(findingCodes(missing), FindingCoordinatesMissing) {
		t.Errorf("codes = %q, want %s", findingCodes(missing), FindingCoordinatesMissing)
	}

	elsewhere := Inspect(Record{
		Location:  Address{Street: "Stationsweg", HouseNr: "6", ZipCode: "3771 VH", City: "Barneveld"},
		Latitude:  "48.8584",
		Longitude: "2.2945",
	})
	if !strings.Contains(findingCodes(elsewhere), FindingCoordinatesOutsideNL) {
		t.Errorf("codes = %q, want %s", findingCodes(elsewhere), FindingCoordinatesOutsideNL)
	}

	here := Inspect(Record{
		Location:  Address{Street: "Utrechtseweg", HouseNr: "232", ZipCode: "6862 AZ", City: "Oosterbeek"},
		Latitude:  "51.98768",
		Longitude: "5.832752",
	})
	if strings.Contains(findingCodes(here), "coordinates") {
		t.Errorf("codes = %q, want no coordinate finding", findingCodes(here))
	}
}

func TestReportsADifferentContactAddressAsANoteRatherThanAProblem(t *testing.T) {
	// A postal address for correspondence may sit somewhere else entirely than
	// the address people visit — a park office, a VVV's own office, the office
	// of a ferry operator. Two different addresses is the normal case, not a
	// defect, so it is reported without asking anybody to repair it.
	got := Inspect(Record{
		Location:    Address{Street: "Apeldoornseweg", HouseNr: "258", ZipCode: "6751 TA", City: "Hoenderloo"},
		ContactInfo: Address{Street: "Houtkampweg", HouseNr: "9", ZipCode: "6731 AV", City: "Otterlo"},
		HasContact:  true,
	})

	var note *Finding
	for i := range got {
		if got[i].Code == FindingContactinfoDiffers {
			note = &got[i]
		}
	}
	if note == nil {
		t.Fatalf("codes = %q, want %s", findingCodes(got), FindingContactinfoDiffers)
	}
	if note.Kind != KindNote {
		t.Errorf("kind = %q, want %q", note.Kind, KindNote)
	}
}

func TestDoesNotCallADifferentlySplitAddressADifference(t *testing.T) {
	// The same address, with the house number inside the street on one side.
	// Comparing field by field made this look like two addresses; it is one.
	got := Inspect(Record{
		Location:    Address{Street: "Cronjéstraat", HouseNr: "15", ZipCode: "6814 AG", City: "Arnhem"},
		ContactInfo: Address{Street: "Cronjéstraat 15", ZipCode: "6814 AG", City: "Arnhem"},
		HasContact:  true,
		Latitude:    "51.99", Longitude: "5.89",
	})

	if strings.Contains(findingCodes(got), FindingContactinfoDiffers) {
		t.Errorf("codes = %q, want no %s", findingCodes(got), FindingContactinfoDiffers)
	}
}

func TestMissingAddressPartsStayProblems(t *testing.T) {
	got := Inspect(Record{Location: Address{City: "Ede"}})

	for _, f := range got {
		if f.Code == FindingStreetMissing && f.Kind != KindProblem {
			t.Errorf("%s has kind %q, want %q", f.Code, f.Kind, KindProblem)
		}
	}
}

func TestDoesNotCallWhitespaceADisagreement(t *testing.T) {
	// The whole point of running the rules first: a trailing space is not drift,
	// it is dirt, and layer one removes it.
	got := Inspect(Record{
		Location:    Address{Street: "Felualaan ", HouseNr: "29", ZipCode: "7313gm", City: "APELDOORN"},
		ContactInfo: Address{Street: "Felualaan", HouseNr: "29", ZipCode: "7313 GM", City: "Apeldoorn"},
		HasContact:  true,
	})

	if strings.Contains(findingCodes(got), FindingContactinfoDiffers) {
		t.Errorf("codes = %q, want no %s", findingCodes(got), FindingContactinfoDiffers)
	}
}

func TestSaysWhatDiffersBetweenTheTwoAddresses(t *testing.T) {
	// Street and house number are compared as one thing, so the detail names
	// that unit rather than pretending the two fields drifted apart separately.
	got := Inspect(Record{
		Location:    Address{Street: "J.C. Wilslaan", HouseNr: "29", ZipCode: "7313 HK", City: "Apeldoorn"},
		ContactInfo: Address{Street: "J.C. Wilslaan", HouseNr: "21", ZipCode: "7313 HK", City: "Apeldoorn"},
		HasContact:  true,
	})

	for _, f := range got {
		if f.Code == FindingContactinfoDiffers {
			if !strings.Contains(f.Detail, "street/housenr") {
				t.Errorf("detail = %q, want it to name street/housenr", f.Detail)
			}
			if strings.Contains(f.Detail, "city") {
				t.Errorf("detail = %q, should not name city", f.Detail)
			}
			return
		}
	}
	t.Fatalf("no %s finding in %q", FindingContactinfoDiffers, findingCodes(got))
}

func TestDoesNotJudgeAForeignAddressByDutchRules(t *testing.T) {
	// A Berlin address has no Dutch postcode and no Dutch coordinates, and saying
	// so every night is noise, not a finding.
	got := Inspect(Record{
		Location:  Address{Street: "Unter den Linden", HouseNr: "1", ZipCode: "10117", City: "Berlin", Country: "DE"},
		Latitude:  "52.5163",
		Longitude: "13.3777",
	})

	if codes := findingCodes(got); codes != "" {
		t.Errorf("codes = %q, want none for a German address", codes)
	}
}

func TestStillReportsAForeignAddressThatIsIncomplete(t *testing.T) {
	got := Inspect(Record{
		Location: Address{City: "Berlin", Country: "DE"},
	})

	if !strings.Contains(findingCodes(got), FindingStreetMissing) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingStreetMissing)
	}
}

func TestReportsAnIncompleteContactCopyOfItsOwn(t *testing.T) {
	// The contact copy is a second address, not a shadow of the first: a copy
	// that carries a street and nothing else is incomplete in its own right.
	got := Inspect(Record{
		Location:    Address{Street: "Utrechtseweg", HouseNr: "232", ZipCode: "6862 AZ", City: "Oosterbeek"},
		ContactInfo: Address{Street: "Utrechtseweg"},
		HasContact:  true,
		Latitude:    "51.98768",
		Longitude:   "5.832752",
	})

	if !strings.Contains(findingCodes(got), FindingContactIncomplete) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingContactIncomplete)
	}
}

func TestSaysNothingAboutAContactCopyThatIsNotThere(t *testing.T) {
	got := Inspect(Record{
		Location:  Address{Street: "Utrechtseweg", HouseNr: "232", ZipCode: "6862 AZ", City: "Oosterbeek"},
		Latitude:  "51.98768",
		Longitude: "5.832752",
	})

	if codes := findingCodes(got); codes != "" {
		t.Errorf("codes = %q, want none", codes)
	}
}
