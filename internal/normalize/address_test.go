package normalize

import "testing"

func TestTrimsAndCollapsesWhitespace(t *testing.T) {
	got, changes := Apply(Address{Street: "IJsselstraat ", City: " Nunspeet", HouseNr: " 12 "})

	if got.Street != "IJsselstraat" {
		t.Errorf("street = %q, want %q", got.Street, "IJsselstraat")
	}
	if got.City != "Nunspeet" {
		t.Errorf("city = %q, want %q", got.City, "Nunspeet")
	}
	if got.HouseNr != "12" {
		t.Errorf("housenr = %q, want %q", got.HouseNr, "12")
	}
	if len(changes) != 3 {
		t.Errorf("changes = %d, want 3: %+v", len(changes), changes)
	}
}

func TestDropsTheLiteralNullHouseNumber(t *testing.T) {
	got, changes := Apply(Address{Street: "Hullerweg", HouseNr: "null"})

	if got.HouseNr != "" {
		t.Errorf("housenr = %q, want empty", got.HouseNr)
	}
	if len(changes) != 1 || changes[0].Rule != RuleHouseNrNull {
		t.Errorf("changes = %+v, want one %s", changes, RuleHouseNrNull)
	}
}

func TestFoldsASingleHouseNumberSuffixOntoTheNumber(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"41a", "41A"},
		{"9B", "9B"},
		{"11 a", "11A"},
		{"6-a", "6A"},
	} {
		got, _ := Apply(Address{HouseNr: tc.in})
		if got.HouseNr != tc.want {
			t.Errorf("housenr %q -> %q, want %q", tc.in, got.HouseNr, tc.want)
		}
	}
}

func TestLeavesAHouseNumberRangeAlone(t *testing.T) {
	// "17-19" is two house numbers, not a number with a suffix. Folding it would
	// invent an address, which is the one thing this pass may never do.
	got, changes := Apply(Address{HouseNr: "17-19"})

	if got.HouseNr != "17-19" {
		t.Errorf("housenr = %q, want %q", got.HouseNr, "17-19")
	}
	if len(changes) != 0 {
		t.Errorf("changes = %+v, want none", changes)
	}
}

func TestFormatsTheZipcode(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"8075re", "8075 RE"},
		{"6511JV", "6511 JV"},
		{"6862  az", "6862 AZ"},
	} {
		got, _ := Apply(Address{ZipCode: tc.in})
		if got.ZipCode != tc.want {
			t.Errorf("zipcode %q -> %q, want %q", tc.in, got.ZipCode, tc.want)
		}
	}
}

func TestLeavesAnUnrecognisableZipcodeAlone(t *testing.T) {
	// "7361" is half a postcode. Completing it is a lookup, not a normalisation.
	got, changes := Apply(Address{ZipCode: "7361"})

	if got.ZipCode != "7361" {
		t.Errorf("zipcode = %q, want %q", got.ZipCode, "7361")
	}
	if len(changes) != 0 {
		t.Errorf("changes = %+v, want none", changes)
	}
}

func TestFixesCityCasingOnlyWhenItIsAllOneCase(t *testing.T) {
	got, _ := Apply(Address{City: "HAARLEM"})
	if got.City != "Haarlem" {
		t.Errorf("city = %q, want %q", got.City, "Haarlem")
	}

	// Anything carrying capitalisation, a dash or an apostrophe of its own is
	// somebody's spelling and is left exactly as found. That guard is narrow on
	// purpose, and it is the same one ff-model applies: "nieuw-lekkerland" is
	// not touched either, because guessing where the capitals go in a compound
	// name is a judgement, not a normalisation.
	for _, in := range []string{"Den Haag", "Voorst - Empe Noord", "'s-Hertogenbosch", "nieuw-lekkerland"} {
		got, changes := Apply(Address{City: in})
		if got.City != in || len(changes) != 0 {
			t.Errorf("city %q -> %q with %+v, want unchanged", in, got.City, changes)
		}
	}
}

func TestIsIdempotent(t *testing.T) {
	first, _ := Apply(Address{Street: "  Houtweg ", HouseNr: "41 a", ZipCode: "8167pj", City: "OENE"})
	second, changes := Apply(first)

	if second != first {
		t.Errorf("second pass changed %+v into %+v", first, second)
	}
	if len(changes) != 0 {
		t.Errorf("second pass reported %+v, want none", changes)
	}
}

func TestReportsWhatItChangedAndWhy(t *testing.T) {
	_, changes := Apply(Address{Street: "Dorpsstraat ", ZipCode: "3882bm"})

	if len(changes) != 2 {
		t.Fatalf("changes = %+v, want 2", changes)
	}
	if changes[0].Field != "street" || changes[0].From != "Dorpsstraat " || changes[0].To != "Dorpsstraat" {
		t.Errorf("first change = %+v", changes[0])
	}
	if changes[1].Rule != RuleZipcodeFormat {
		t.Errorf("second change rule = %q, want %q", changes[1].Rule, RuleZipcodeFormat)
	}
}

func TestKeepsTheDutchIJDigraphIntact(t *testing.T) {
	// "IJmuiden" is one letter written as two, and lowercasing the J is a
	// spelling mistake, not a normalisation. ff-model's own rule gets this
	// wrong; this package refuses to copy that.
	for _, tc := range []struct{ in, want string }{
		{"IJMUIDEN", "IJmuiden"},
		{"ijmuiden", "IJmuiden"},
		{"IJSSELSTEIN", "IJsselstein"},
		{"IJZENDOORN", "IJzendoorn"},
		{"OOSTERBEEK", "Oosterbeek"},
		{"NIJMEGEN", "Nijmegen"},
	} {
		got, _ := Apply(Address{City: tc.in})
		if got.City != tc.want {
			t.Errorf("city %q -> %q, want %q", tc.in, got.City, tc.want)
		}
	}
}

func TestCollapsesWhitespaceThatIsNotASpace(t *testing.T) {
	// A non-breaking space is what a paste out of a CMS leaves behind, and Go's
	// \s does not match it.
	got, changes := Apply(Address{ZipCode: "6862 az", City: "Oosterbeek "})

	if got.ZipCode != "6862 AZ" {
		t.Errorf("zipcode = %q, want %q", got.ZipCode, "6862 AZ")
	}
	if got.City != "Oosterbeek" {
		t.Errorf("city = %q, want %q", got.City, "Oosterbeek")
	}
	if len(changes) == 0 {
		t.Error("changes = none, want the whitespace to be reported")
	}
}
