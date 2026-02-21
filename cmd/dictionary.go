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
	EntityType   string           `json:"entityType"`
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

	idStr := ""
	if cat.ID != nil && *cat.ID != "" {
		idStr = fmt.Sprintf(" (ID: %s)", *cat.ID)
	}

	deprecated := ""
	if cat.Deprecated != nil && cat.Deprecated != false {
		if s, ok := cat.Deprecated.(string); ok && s != "" {
			deprecated = " [DEPRECATED]"
		} else if b, ok := cat.Deprecated.(bool); ok && b {
			deprecated = " [DEPRECATED]"
		}
	}

	fmt.Printf("%s%s  %s%s%s\n", indent, cat.CnetID, cat.Name, idStr, deprecated)

	for _, child := range cat.Children {
		printCategory(child, depth+1)
	}
}

type DictionaryCategoriesCmd struct {
	JSON bool   `short:"j" help:"Output as JSON."`
	Lang string `name:"lang" default:"nl" help:"Language for category labels (nl, en, de). Default: nl."`
	Type string `name:"type" short:"t" default:"" help:"Filter by entity type: event, location, route, eventgroup." enum:",event,location,route,eventgroup"`
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

	// Map user-friendly type names to ontology entityType values
	entityTypeMap := map[string]string{
		"event":      "EVENEMENT",
		"location":   "LOCATIE",
		"route":      "ROUTE",
		"eventgroup": "EVENEMENTGROEP",
	}
	filterEntityType := entityTypeMap[c.Type]

	// Filter top-level categorizations by entity type if specified
	topCats := ontology.Categorizations
	if filterEntityType != "" {
		var filtered []Categorization
		for _, cat := range topCats {
			if cat.EntityType == filterEntityType {
				filtered = append(filtered, cat)
			}
		}
		topCats = filtered
	}

	// Collect all leaf categories
	type flatCat struct {
		ID     string `json:"id"`
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

			if len(cat.Children) > 0 {
				collect(cat.Children, label)
			} else if cat.CnetID != "" {
				// Leaf category: no children, use cnetID as the ID
				id := cat.CnetID
				if cat.ID != nil && *cat.ID != "" {
					id = *cat.ID
				}
				categories = append(categories, flatCat{
					ID:     id,
					Label:  label,
					Parent: parent,
				})
			}
		}
	}
	collect(topCats, "")

	if c.JSON {
		return printJSON(categories)
	}

	if len(categories) == 0 {
		fmt.Println("No categories found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLABEL\tPARENT")
	fmt.Fprintln(w, "--\t-----\t------")

	for _, cat := range categories {
		fmt.Fprintf(w, "%s\t%s\t%s\n", cat.ID, truncate(cat.Label, 40), truncate(cat.Parent, 30))
	}
	w.Flush()

	fmt.Printf("\nTotal: %d categories\n", len(categories))
	return nil
}
