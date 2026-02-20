package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type DictionaryCmd struct {
	Keywords   DictionaryKeywordsCmd   `cmd:"" help:"List keywords for a resource type. Keywords are used to tag and categorize resources. Stored per account."`
	Markers    DictionaryMarkersCmd    `cmd:"" help:"List markers for a resource type. Markers are used for internal labeling and filtering. Stored per account."`
	Ontology   DictionaryOntologyCmd   `cmd:"" help:"Show the categorization ontology (category tree). Returns the full hierarchy of categories and their translations."`
	Categories DictionaryCategoriesCmd `cmd:"" help:"List all leaf categories from the ontology with their IDs. Useful for finding category IDs to use with --categories filter."`
}

type DictionaryKeywordsCmd struct {
	Type string `arg:"" help:"Resource type. Valid types: event, eventGroup, route, location, venue." enum:"event,eventGroup,route,location,venue"`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *DictionaryKeywordsCmd) Run(client *api.Client) error {
	data, err := client.GetKeywords(c.Type)
	if err != nil {
		return err
	}

	return printRawJSON(data)
}

type DictionaryMarkersCmd struct {
	Type string `arg:"" help:"Resource type. Valid types: event, eventGroup, route, location, venue." enum:"event,eventGroup,route,location,venue"`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *DictionaryMarkersCmd) Run(client *api.Client) error {
	data, err := client.GetMarkers(c.Type)
	if err != nil {
		return err
	}

	return printRawJSON(data)
}

type DictionaryOntologyCmd struct {
	JSON bool `short:"j" help:"Output as JSON."`
}

func (c *DictionaryOntologyCmd) Run(client *api.Client) error {
	data, err := client.GetOntology()
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(data)
	}

	// Parse and display as a tree
	var ontology struct {
		LastModified    string           `json:"lastModified"`
		Categorizations []Categorization `json:"categorizations"`
	}
	if err := json.Unmarshal(data, &ontology); err != nil {
		// Fall back to raw JSON on parse error
		return printRawJSON(data)
	}

	if ontology.LastModified != "" {
		fmt.Printf("Last modified: %s\n\n", ontology.LastModified)
	}

	for _, cat := range ontology.Categorizations {
		printCategory(cat, 0)
	}

	return nil
}

type Categorization struct {
	CnetID       string           `json:"cnetID"`
	Name         string           `json:"categorization"`
	ID           *string          `json:"categorizationId"`
	Deprecated   interface{}      `json:"deprecated"`
	Children     []Categorization `json:"child"`
	Translations []CatTranslation `json:"categoryTranslations"`
}

type CatTranslation struct {
	Lang  string `json:"lang"`
	Label string `json:"label"`
}

func printCategory(cat Categorization, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	id := "-"
	if cat.ID != nil && *cat.ID != "" {
		id = *cat.ID
	}

	deprecated := ""
	if cat.Deprecated != nil && cat.Deprecated != false {
		if s, ok := cat.Deprecated.(string); ok && s != "" {
			deprecated = " [DEPRECATED]"
		} else if b, ok := cat.Deprecated.(bool); ok && b {
			deprecated = " [DEPRECATED]"
		}
	}

	fmt.Printf("%s%s  %s (ID: %s)%s\n", indent, cat.CnetID, cat.Name, id, deprecated)

	for _, child := range cat.Children {
		printCategory(child, depth+1)
	}
}

type DictionaryCategoriesCmd struct {
	JSON bool `short:"j" help:"Output as JSON."`
	Lang string `name:"lang" default:"nl" help:"Language for category labels (nl, en, de). Default: nl."`
}

func (c *DictionaryCategoriesCmd) Run(client *api.Client) error {
	data, err := client.GetOntology()
	if err != nil {
		return err
	}

	var ontology struct {
		Categorizations []Categorization `json:"categorizations"`
	}
	if err := json.Unmarshal(data, &ontology); err != nil {
		return printRawJSON(data)
	}

	// Collect all leaf categories (those with a categorizationId)
	type flatCat struct {
		ID     string `json:"id"`
		CnetID string `json:"cnetId"`
		Label  string `json:"label"`
		Parent string `json:"parent"`
	}

	var categories []flatCat
	var collect func(cats []Categorization, parent string)
	collect = func(cats []Categorization, parent string) {
		for _, cat := range cats {
			label := cat.Name
			for _, t := range cat.Translations {
				if t.Lang == c.Lang {
					label = t.Label
					break
				}
			}

			if cat.ID != nil && *cat.ID != "" {
				categories = append(categories, flatCat{
					ID:     *cat.ID,
					CnetID: cat.CnetID,
					Label:  label,
					Parent: parent,
				})
			}

			if len(cat.Children) > 0 {
				collect(cat.Children, label)
			}
		}
	}
	collect(ontology.Categorizations, "")

	if c.JSON {
		return printJSON(categories)
	}

	if len(categories) == 0 {
		fmt.Println("No categories found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCNET_ID\tLABEL\tPARENT")
	fmt.Fprintln(w, "--\t-------\t-----\t------")

	for _, cat := range categories {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cat.ID, cat.CnetID, truncate(cat.Label, 40), truncate(cat.Parent, 30))
	}
	w.Flush()

	fmt.Printf("\nTotal: %d categories\n", len(categories))
	return nil
}
