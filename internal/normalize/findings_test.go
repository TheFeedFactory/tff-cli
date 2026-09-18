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

func TestReportsTheTwoAddressCopiesDisagreeing(t *testing.T) {
	got := Inspect(Record{
		Location:    Address{Street: "Felualaan", HouseNr: "29", ZipCode: "7313 GM", City: "Apeldoorn"},
		ContactInfo: Address{Street: "Hoofdstraat", HouseNr: "1", ZipCode: "7311 KA", City: "Apeldoorn"},
		HasContact:  true,
	})

	if !strings.Contains(findingCodes(got), FindingCopiesDiffer) {
		t.Errorf("codes = %q, want %s", findingCodes(got), FindingCopiesDiffer)
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

	if strings.Contains(findingCodes(got), FindingCopiesDiffer) {
		t.Errorf("codes = %q, want no %s", findingCodes(got), FindingCopiesDiffer)
	}
}

func TestSaysWhichFieldsDisagree(t *testing.T) {
	got := Inspect(Record{
		Location:    Address{Street: "Felualaan", HouseNr: "29", ZipCode: "7313 GM", City: "Apeldoorn"},
		ContactInfo: Address{Street: "Felualaan", HouseNr: "31", ZipCode: "7313 GM", City: "Apeldoorn"},
		HasContact:  true,
	})

	for _, f := range got {
		if f.Code == FindingCopiesDiffer {
			if !strings.Contains(f.Detail, "housenr") {
				t.Errorf("detail = %q, want it to name housenr", f.Detail)
			}
			if strings.Contains(f.Detail, "street") {
				t.Errorf("detail = %q, should not name street", f.Detail)
			}
			return
		}
	}
	t.Fatalf("no %s finding in %q", FindingCopiesDiffer, findingCodes(got))
}
