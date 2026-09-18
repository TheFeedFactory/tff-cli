package cmd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

// pagesOf serves a fixed catalogue in pages, and records what it was asked for.
func pagesOf(total, perPage int, asked *[]int) listPage {
	return func(opts api.ListOptions) (*api.SearchResult, error) {
		*asked = append(*asked, opts.Page)
		start := opts.Page * perPage
		if start >= total {
			return &api.SearchResult{Hits: total, Page: opts.Page}, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		var results []json.RawMessage
		for i := start; i < end; i++ {
			results = append(results, json.RawMessage(fmt.Sprintf(`{"id":"loc-%d"}`, i)))
		}
		return &api.SearchResult{Hits: total, Page: opts.Page, Results: results}, nil
	}
}

func TestWalksEveryPageOfTheSelection(t *testing.T) {
	var asked []int
	var seen []string

	err := eachLocation(pagesOf(25, 10, &asked), api.ListOptions{}, 0, func(r api.Resource) {
		seen = append(seen, r.ID)
	})
	if err != nil {
		t.Fatalf("eachLocation: %v", err)
	}

	if len(seen) != 25 {
		t.Errorf("visited %d locations, want 25", len(seen))
	}
	if seen[0] != "loc-0" || seen[24] != "loc-24" {
		t.Errorf("visited %q first and %q last", seen[0], seen[24])
	}
	if len(asked) != 3 {
		t.Errorf("asked for pages %v, want 3 requests", asked)
	}
}

func TestStopsAtTheLimitWithoutAskingForMore(t *testing.T) {
	var asked []int
	var seen []string

	err := eachLocation(pagesOf(1000, 10, &asked), api.ListOptions{}, 12, func(r api.Resource) {
		seen = append(seen, r.ID)
	})
	if err != nil {
		t.Fatalf("eachLocation: %v", err)
	}

	if len(seen) != 12 {
		t.Errorf("visited %d locations, want 12", len(seen))
	}
	if len(asked) != 2 {
		t.Errorf("asked for pages %v, want to stop after 2", asked)
	}
}

func TestStopsOnAnEmptyPageEvenWhenHitsDisagrees(t *testing.T) {
	// A count that overstates what the index will hand out must not become an
	// endless walk.
	var asked []int
	err := eachLocation(func(opts api.ListOptions) (*api.SearchResult, error) {
		asked = append(asked, opts.Page)
		if opts.Page == 0 {
			return &api.SearchResult{Hits: 9999, Results: []json.RawMessage{json.RawMessage(`{"id":"only"}`)}}, nil
		}
		return &api.SearchResult{Hits: 9999}, nil
	}, api.ListOptions{}, 0, func(api.Resource) {})
	if err != nil {
		t.Fatalf("eachLocation: %v", err)
	}

	if len(asked) != 2 {
		t.Errorf("asked for pages %v, want to stop after the empty one", asked)
	}
}

func TestReportsTheErrorFromAPageInsteadOfSwallowingIt(t *testing.T) {
	err := eachLocation(func(api.ListOptions) (*api.SearchResult, error) {
		return nil, fmt.Errorf("result window exceeded")
	}, api.ListOptions{}, 0, func(api.Resource) {})

	if err == nil {
		t.Fatal("eachLocation returned nil, want the page error")
	}
}

func TestCollapsesEveryRuleOnOneFieldIntoOneRewrite(t *testing.T) {
	entry := inspectLocation(api.Resource{
		Location: &api.Location{Address: &api.Address{HouseNr: " 41 a ", City: "OENE"}},
	})

	collapsed := collapseByField(entry.Changes)
	var housenr *fieldChange
	for i := range collapsed {
		if collapsed[i].Field == "location.address.housenr" {
			housenr = &collapsed[i]
		}
	}
	if housenr == nil {
		t.Fatalf("no housenr change in %+v", collapsed)
	}
	if housenr.From != " 41 a " || housenr.To != "41A" {
		t.Errorf("housenr %q -> %q, want %q -> %q", housenr.From, housenr.To, " 41 a ", "41A")
	}
	if housenr.Rules != "whitespace+housenr-suffix" {
		t.Errorf("rules = %q, want both", housenr.Rules)
	}
}
