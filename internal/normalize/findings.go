package normalize

import (
	"fmt"
	"strconv"
	"strings"
)

// Finding is something wrong with a record that the rules may not fix by
// themselves: a value that is absent, a value that is unrecognisable, or two
// copies of the address that disagree. Each one is work for a person or for a
// lookup, and naming it is the whole job of the dry run.
const (
	FindingStreetMissing       = "street-missing"
	FindingHouseNrMissing      = "housenr-missing"
	FindingZipcodeMissing      = "zipcode-missing"
	FindingCityMissing         = "city-missing"
	FindingZipcodeUnrecognised = "zipcode-unrecognised"
	FindingCoordinatesMissing  = "coordinates-missing"
	FindingCoordinatesOutside  = "coordinates-outside-nl"
	FindingCopiesDiffer        = "copies-differ"
)

// FindingCoordinatesOutsideNL is the spelling used at the call sites.
const FindingCoordinatesOutsideNL = FindingCoordinatesOutside

// The Netherlands with room to spare: anything outside this is either a foreign
// address or a coordinate written the wrong way round, and both need a person.
const (
	minLon, maxLon = 3.0, 7.5
	minLat, maxLat = 50.5, 53.8
)

// Record is one location's two copies of its address plus its coordinates.
// HasContact distinguishes "no contact address at all" from "an empty one", so a
// record that simply has one copy is not reported as two copies disagreeing.
type Record struct {
	Location    Address
	ContactInfo Address
	HasContact  bool
	Latitude    string
	Longitude   string
}

// Finding is one problem, with enough detail to act on it.
type Finding struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// Inspect reports what the deterministic rules cannot repair. It compares the
// two address copies *after* normalising both, so that dirt is not mistaken for
// disagreement.
func Inspect(r Record) []Finding {
	var findings []Finding
	add := func(code, detail string) {
		findings = append(findings, Finding{Code: code, Detail: detail})
	}

	location, _ := Apply(r.Location)

	if location.Street == "" {
		add(FindingStreetMissing, "")
	}
	if location.HouseNr == "" {
		add(FindingHouseNrMissing, "")
	}
	if location.City == "" {
		add(FindingCityMissing, "")
	}
	switch {
	case location.ZipCode == "":
		add(FindingZipcodeMissing, "")
	case !dutchZipcode.MatchString(location.ZipCode):
		add(FindingZipcodeUnrecognised, location.ZipCode)
	}

	findings = append(findings, inspectCoordinates(r)...)

	if r.HasContact {
		contact, _ := Apply(r.ContactInfo)
		if differing := differingFields(location, contact); len(differing) > 0 {
			add(FindingCopiesDiffer, strings.Join(differing, ", "))
		}
	}

	return findings
}

func inspectCoordinates(r Record) []Finding {
	if strings.TrimSpace(r.Latitude) == "" || strings.TrimSpace(r.Longitude) == "" {
		return []Finding{{Code: FindingCoordinatesMissing}}
	}

	lat, latErr := strconv.ParseFloat(strings.TrimSpace(r.Latitude), 64)
	lon, lonErr := strconv.ParseFloat(strings.TrimSpace(r.Longitude), 64)
	if latErr != nil || lonErr != nil {
		return []Finding{{Code: FindingCoordinatesMissing, Detail: "unreadable"}}
	}
	if lat < minLat || lat > maxLat || lon < minLon || lon > maxLon {
		return []Finding{{Code: FindingCoordinatesOutside, Detail: fmt.Sprintf("%v, %v", lat, lon)}}
	}
	return nil
}

// differingFields names the fields on which the two copies disagree, ignoring a
// field one copy simply does not carry: a contact address that omits the
// postcode is incomplete, not contradictory, and is already reported as such.
func differingFields(location, contact Address) []string {
	var differing []string
	for _, f := range []struct {
		name     string
		lhs, rhs string
	}{
		{"street", location.Street, contact.Street},
		{"housenr", location.HouseNr, contact.HouseNr},
		{"zipcode", location.ZipCode, contact.ZipCode},
		{"city", location.City, contact.City},
	} {
		if f.lhs == "" || f.rhs == "" {
			continue
		}
		if !strings.EqualFold(f.lhs, f.rhs) {
			differing = append(differing, f.name)
		}
	}
	return differing
}
