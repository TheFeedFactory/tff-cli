package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
	"github.com/TheFeedFactory/tff-cli/internal/normalize"
)

// normalizePageSize is what one request asks for while walking a selection. The
// API accepts more, but a smaller page fails sooner and retries cheaper.
const normalizePageSize = 200

type LocationsNormalizeCmd struct {
	Markers      string `help:"Comma-separated markers filter. Prefix with '!' to exclude."`
	Keywords     string `help:"Comma-separated keywords filter."`
	Published    string `help:"Filter by published state (true/false)."`
	UserOrg      string `name:"userorganisation" help:"Filter by user organisation."`
	Search       string `short:"s" help:"Full-text search query."`
	UpdatedSince string `name:"updated-since" help:"Items updated after date. Relative: 2w, 3d, 1mo, 1y. Absolute: 2026-01-15."`
	Limit        int    `help:"Stop after this many locations (0 = all of them)."`

	Details bool   `help:"List every affected location instead of a few examples per rule."`
	CSV     string `help:"Write one row per proposed change to this file, as a worklist."`
	JSON    bool   `short:"j" help:"Output the full report as JSON."`
}

// locationReport is one location's proposed changes and remaining problems.
type locationReport struct {
	ID       string              `json:"id"`
	Title    string              `json:"title"`
	City     string              `json:"city"`
	Changes  []normalize.Change  `json:"changes,omitempty"`
	Findings []normalize.Finding `json:"findings,omitempty"`
}

type normalizeReport struct {
	DryRun        bool             `json:"dryRun"`
	Selection     string           `json:"selection"`
	Inspected     int              `json:"inspected"`
	WouldChange   int              `json:"wouldChange"`
	WithFindings  int              `json:"withFindings"`
	NeedsAPerson  int              `json:"needsAPerson"`
	Clean         int              `json:"clean"`
	ChangesByRule map[string]int   `json:"changesByRule"`
	FindingCounts map[string]int   `json:"findingCounts"`
	NotIdempotent []string         `json:"notIdempotent,omitempty"`
	Locations     []locationReport `json:"locations,omitempty"`
}

func (c *LocationsNormalizeCmd) Run(client *api.Client) error {
	opts := api.ListOptions{
		Markers:   c.Markers,
		Keywords:  c.Keywords,
		Published: c.Published,
		UserOrg:   c.UserOrg,
		Search:    c.Search,
		Size:      normalizePageSize,

		// A stable order for the walk. The default order is by last
		// modification, and a feed import bumps that while we are paging, which
		// silently drops records out of one page and repeats them in another.
		// Creation date never moves.
		Sort: "created",
		Asc:  true,
	}
	if c.UpdatedSince != "" {
		iso, err := ParseRelativeISO(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("--updated-since: %w", err)
		}
		opts.UpdatedSince = iso
	}

	report := normalizeReport{
		DryRun:        true,
		Selection:     describeSelection(c),
		ChangesByRule: map[string]int{},
		FindingCounts: map[string]int{},
	}

	err := eachLocation(client.ListLocations, opts, c.Limit, func(r api.Resource) {
		report.Inspected++
		entry := inspectLocation(r)

		for _, ch := range entry.Changes {
			report.ChangesByRule[ch.Rule]++
		}
		for _, f := range entry.Findings {
			report.FindingCounts[f.Code]++
		}
		if len(entry.Changes) > 0 {
			report.WouldChange++
		}
		if len(entry.Findings) > 0 {
			report.WithFindings++
		}
		if needsAPerson(entry.Findings) {
			report.NeedsAPerson++
		}
		if len(entry.Changes) == 0 && len(entry.Findings) == 0 {
			report.Clean++
		}

		// A second pass over the proposed values must find nothing left to do.
		// If it does, a rule is fighting another rule and the report is the only
		// place that would ever show it.
		if !isSettled(r) {
			report.NotIdempotent = append(report.NotIdempotent, entry.ID)
		}

		if len(entry.Changes) > 0 || len(entry.Findings) > 0 {
			report.Locations = append(report.Locations, entry)
		}
	})
	if err != nil {
		return err
	}

	if c.CSV != "" {
		if err := writeChangeCSV(c.CSV, report.Locations); err != nil {
			return err
		}
	}

	if c.JSON {
		return printJSON(report)
	}
	printReport(report, c.Details)
	if c.CSV != "" {
		fmt.Printf("\nWorklist written to %s\n", c.CSV)
	}
	return nil
}

// inspectLocation runs the rules over both copies of one location's address.
// The changes are reported per copy, because a later pass that writes them has
// to write them in two places.
func inspectLocation(r api.Resource) locationReport {
	location := addressOf(r.Location)
	contact, hasContact := contactAddressOf(r.ContactInfo)

	_, locationChanges := normalize.Apply(location)
	entry := locationReport{ID: r.ID, Title: r.GetTitle(), City: location.City}
	entry.Changes = prefixed("location.address", locationChanges)

	if hasContact {
		_, contactChanges := normalize.Apply(contact)
		entry.Changes = append(entry.Changes, prefixed("contactinfo.address", contactChanges)...)
	}

	lat, lon := coordinatesOf(r.Location)
	entry.Findings = normalize.Inspect(normalize.Record{
		Location:    location,
		ContactInfo: contact,
		HasContact:  hasContact,
		Latitude:    lat,
		Longitude:   lon,
	})
	return entry
}

// isSettled reports whether normalising the already-normalised values changes
// nothing further, for both copies of the address.
func isSettled(r api.Resource) bool {
	for _, a := range []normalize.Address{addressOf(r.Location), contactAddress(r.ContactInfo)} {
		once, _ := normalize.Apply(a)
		if _, again := normalize.Apply(once); len(again) > 0 {
			return false
		}
	}
	return true
}

func contactAddress(ci *api.ContactInfo) normalize.Address {
	a, _ := contactAddressOf(ci)
	return a
}

// needsAPerson reports whether a finding is a hole in the address rather than an
// observation about it. A hole cannot be filled by any rule in this package.
func needsAPerson(findings []normalize.Finding) bool {
	for _, f := range findings {
		switch f.Code {
		case normalize.FindingStreetMissing, normalize.FindingHouseNrMissing,
			normalize.FindingZipcodeMissing, normalize.FindingCityMissing:
			return true
		}
	}
	return false
}

func prefixed(copyName string, changes []normalize.Change) []normalize.Change {
	out := make([]normalize.Change, 0, len(changes))
	for _, ch := range changes {
		ch.Field = copyName + "." + ch.Field
		out = append(out, ch)
	}
	return out
}

func addressOf(l *api.Location) normalize.Address {
	if l == nil || l.Address == nil {
		return normalize.Address{}
	}
	return fromAPI(l.Address)
}

func contactAddressOf(ci *api.ContactInfo) (normalize.Address, bool) {
	if ci == nil || ci.Address == nil {
		return normalize.Address{}, false
	}
	return fromAPI(ci.Address), true
}

func fromAPI(a *api.Address) normalize.Address {
	return normalize.Address{
		Street:  a.Street,
		HouseNr: a.HouseNr,
		ZipCode: a.ZipCode,
		City:    a.City,
		Country: a.Country,
	}
}

func coordinatesOf(l *api.Location) (lat, lon string) {
	if l == nil || l.Address == nil || len(l.Address.GisCoordinates) == 0 {
		return "", ""
	}
	first := l.Address.GisCoordinates[0]
	return first.YCoordinate, first.XCoordinate
}

// listPage is one page of a selection. Taking the walk's dependency as a
// function rather than the client is what lets the paging be tested, which is
// where the risk in this command actually sits.
type listPage func(api.ListOptions) (*api.SearchResult, error)

// eachLocation walks the whole selection, page by page, and stops early at limit.
func eachLocation(list listPage, opts api.ListOptions, limit int, visit func(api.Resource)) error {
	seen := 0
	for page := 0; ; page++ {
		opts.Page = page
		result, err := list(opts)
		if err != nil {
			return err
		}
		resources, err := api.ParseResources(result.Results)
		if err != nil {
			return err
		}
		if len(resources) == 0 {
			return nil
		}
		for _, r := range resources {
			visit(r)
			seen++
			if limit > 0 && seen >= limit {
				return nil
			}
		}
		if seen >= result.Hits {
			return nil
		}
	}
}

func describeSelection(c *LocationsNormalizeCmd) string {
	var parts []string
	for _, p := range []struct{ name, value string }{
		{"markers", c.Markers},
		{"keywords", c.Keywords},
		{"published", c.Published},
		{"userorganisation", c.UserOrg},
		{"search", c.Search},
	} {
		if p.value != "" {
			parts = append(parts, p.name+"="+p.value)
		}
	}
	if c.UpdatedSince != "" {
		parts = append(parts, "updated-since="+c.UpdatedSince)
	}
	if c.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit=%d", c.Limit))
	}
	if len(parts) == 0 {
		return "all locations"
	}
	return strings.Join(parts, " ")
}

func printReport(report normalizeReport, details bool) {
	fmt.Printf("Dry run over %s — nothing is written.\n\n", report.Selection)
	fmt.Printf("Inspected            %d locations\n", report.Inspected)
	fmt.Printf("Would be rewritten   %d  (deterministic, safe to apply)\n", report.WouldChange)
	fmt.Printf("Has findings         %d  (of which %d are missing an address part)\n", report.WithFindings, report.NeedsAPerson)
	fmt.Printf("Already clean        %d\n", report.Clean)

	printCounts("\nProposed changes, counted per rewritten value (safe to write)", report.ChangesByRule)
	printCounts("\nFindings, counted per location (these need a lookup or a person)", report.FindingCounts)

	if len(report.NotIdempotent) > 0 {
		fmt.Printf("\nWARNING: %d locations do not settle after one pass: %s\n",
			len(report.NotIdempotent), strings.Join(report.NotIdempotent, ", "))
	}

	if details {
		printAll(report.Locations)
		return
	}
	printExamples(report.Locations)
}

func printCounts(heading string, counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	fmt.Println(heading)
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	// Count first, then name, so that two runs of the same selection produce
	// byte-identical reports and a diff between them means something.
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, k := range keys {
		fmt.Fprintf(w, "  %s\t%d\n", k, counts[k])
	}
	w.Flush()
}

// printExamples shows the first few locations per rule, which is enough to judge
// whether a rule is doing what it should before anybody runs it for real.
func printExamples(locations []locationReport) {
	const perRule = 3
	shown := map[string]int{}

	fmt.Println("\nExamples")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, l := range locations {
		for _, ch := range l.Changes {
			if shown[ch.Rule] >= perRule {
				continue
			}
			shown[ch.Rule]++
			fmt.Fprintf(w, "  %s\t%s\t%q -> %q\t%s\n",
				ch.Rule, truncate(l.Title, 34), ch.From, ch.To, ch.Field)
		}
	}
	w.Flush()
	fmt.Println("\nRun with --details for every location, or --csv <file> for a worklist.")
}

func printAll(locations []locationReport) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nID\tTITLE\tFIELD\tFROM\tTO\tFINDINGS")
	for _, l := range locations {
		findings := findingCodes(l.Findings)
		if len(l.Changes) == 0 {
			fmt.Fprintf(w, "%s\t%s\t\t\t\t%s\n", l.ID, truncate(l.Title, 34), findings)
			continue
		}
		for _, ch := range collapseByField(l.Changes) {
			fmt.Fprintf(w, "%s\t%s\t%s\t%q\t%q\t%s\n",
				l.ID, truncate(l.Title, 34), ch.Field, ch.From, ch.To, findings)
			findings = ""
		}
	}
	w.Flush()
}

func findingCodes(findings []normalize.Finding) string {
	var codes []string
	for _, f := range findings {
		if f.Detail != "" {
			codes = append(codes, f.Code+"("+f.Detail+")")
			continue
		}
		codes = append(codes, f.Code)
	}
	return strings.Join(codes, " ")
}

// fieldChange is every rule that touched one field, collapsed into the one
// rewrite a person would actually make. Reporting the intermediate value of a
// field two rules touched ("41 a" between the trim and the fold) would show a
// value that appears nowhere in the record and nowhere in the result.
type fieldChange struct {
	Field string
	Rules string
	From  string
	To    string
}

func collapseByField(changes []normalize.Change) []fieldChange {
	var order []string
	byField := map[string]*fieldChange{}

	for _, ch := range changes {
		existing, seen := byField[ch.Field]
		if !seen {
			order = append(order, ch.Field)
			byField[ch.Field] = &fieldChange{Field: ch.Field, Rules: ch.Rule, From: ch.From, To: ch.To}
			continue
		}
		existing.Rules += "+" + ch.Rule
		existing.To = ch.To
	}

	collapsed := make([]fieldChange, 0, len(order))
	for _, field := range order {
		collapsed = append(collapsed, *byField[field])
	}
	return collapsed
}

func writeChangeCSV(path string, locations []locationReport) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer func() {
		// A close error on a file we wrote is a write that did not land, so it
		// may not be swallowed by a successful return.
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", path, closeErr)
		}
	}()

	w := csv.NewWriter(file)
	if err := w.Write([]string{"id", "title", "field", "rules", "from", "to", "findings"}); err != nil {
		return err
	}
	for _, l := range locations {
		findings := findingCodes(l.Findings)
		changes := collapseByField(l.Changes)
		if len(changes) == 0 {
			if err := w.Write([]string{l.ID, l.Title, "", "", "", "", findings}); err != nil {
				return err
			}
			continue
		}
		for _, ch := range changes {
			if err := w.Write([]string{l.ID, l.Title, ch.Field, ch.Rules, ch.From, ch.To, findings}); err != nil {
				return err
			}
		}
	}

	w.Flush()
	return w.Error()
}
