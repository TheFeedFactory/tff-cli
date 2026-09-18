// Package normalize holds the deterministic half of address cleanup: the rules
// that can be applied without asking anybody or anything.
//
// The line it does not cross is the one that matters. Every rule here rewrites a
// value that is already in the record into the same value written properly. None
// of them fills in a value that is missing, because the only sources for that are
// a lookup or a guess, and a guessed address is indistinguishable from a curated
// one the moment it is written. Missing values are reported, never invented.
//
// Zipcode and city casing deliberately mirror Address.normaliseAdresItems in
// ff-model, so that a later pass that writes these values back cannot disagree
// with the model's own normalisation.
package normalize

import (
	"regexp"
	"strings"
)

// Rule names the reason a value changed, so a report can be read per rule rather
// than per field.
const (
	RuleWhitespace    = "whitespace"
	RuleHouseNrNull   = "housenr-null"
	RuleHouseNrSuffix = "housenr-suffix"
	RuleZipcodeFormat = "zipcode-format"
	RuleCityCase      = "city-case"
)

// Address is the part of a postal address these rules touch.
type Address struct {
	Street  string
	HouseNr string
	ZipCode string
	City    string
	Country string
}

// Change is one field rewritten by one rule.
type Change struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	From  string `json:"from"`
	To    string `json:"to"`
}

var (
	// Go's \s is ASCII only, and the commonest dirt in a pasted address is a
	// non-breaking space, so those are named explicitly.
	whitespaceRun = regexp.MustCompile("[\\s\u00a0\u2007\u202f]+")
	// A number followed by exactly one letter, however the source spelled the
	// join. Anything longer is not a suffix, and a second number is a range.
	houseNrSuffix = regexp.MustCompile(`^(\d+)\s*[-/ ]?\s*([A-Za-z])$`)
	dutchZipcode  = regexp.MustCompile(`^(\d{4})\s*([A-Za-z]{2})$`)
	// ff-model touches a city name only when it carries no capitalisation of its
	// own, which is what keeps "Den Haag" and "Voorst - Empe Noord" intact.
	singleCaseCity = regexp.MustCompile(`^([a-z\s]+|[A-Z\s]+)$`)
	wordStart      = regexp.MustCompile(`(^|[\s])([a-z])`)
	// IJ is one Dutch letter written as two, and it capitalises as a pair.
	dutchIJ = regexp.MustCompile(`(^|[\s])Ij`)
)

// Apply returns the address as it should be written, plus every change it made.
// It is idempotent: applying it to its own output yields no further changes.
func Apply(a Address) (Address, []Change) {
	var changes []Change

	record := func(field, rule, from, to string) {
		if from != to {
			changes = append(changes, Change{Field: field, Rule: rule, From: from, To: to})
		}
	}

	for _, f := range []struct {
		name string
		val  *string
	}{
		{"street", &a.Street},
		{"housenr", &a.HouseNr},
		{"zipcode", &a.ZipCode},
		{"city", &a.City},
		{"country", &a.Country},
	} {
		tidy := collapseWhitespace(*f.val)
		record(f.name, RuleWhitespace, *f.val, tidy)
		*f.val = tidy
	}

	if strings.EqualFold(a.HouseNr, "null") {
		// Written by a connector that composed a null into a string; it is not a
		// house number and never was.
		record("housenr", RuleHouseNrNull, a.HouseNr, "")
		a.HouseNr = ""
	}

	if m := houseNrSuffix.FindStringSubmatch(a.HouseNr); m != nil {
		folded := m[1] + strings.ToUpper(m[2])
		record("housenr", RuleHouseNrSuffix, a.HouseNr, folded)
		a.HouseNr = folded
	}

	if m := dutchZipcode.FindStringSubmatch(a.ZipCode); m != nil {
		formatted := m[1] + " " + strings.ToUpper(m[2])
		record("zipcode", RuleZipcodeFormat, a.ZipCode, formatted)
		a.ZipCode = formatted
	}

	if a.City != "" && singleCaseCity.MatchString(a.City) {
		titled := titleCase(a.City)
		record("city", RuleCityCase, a.City, titled)
		a.City = titled
	}

	return a, changes
}

func collapseWhitespace(value string) string {
	return strings.TrimSpace(whitespaceRun.ReplaceAllString(value, " "))
}

// titleCase capitalises the first letter of every word. It mirrors ff-model's
// Address.normaliseAdresItems with one deliberate difference: ff-model lowercases
// the J of the IJ digraph ("IJmuiden" becomes "Ijmuiden"), which is a spelling
// mistake rather than a normalisation, and this package will not write one.
func titleCase(value string) string {
	lowered := strings.ToLower(value)
	titled := wordStart.ReplaceAllStringFunc(lowered, strings.ToUpper)
	return dutchIJ.ReplaceAllString(titled, "${1}IJ")
}
