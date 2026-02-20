package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type LocationsCmd struct {
	List      LocationsListCmd      `cmd:"" help:"List and search locations. Supports full-text search, workflow status filtering, markers, keywords, and more."`
	Get       LocationsGetCmd       `cmd:"" help:"Get detailed information about a specific location by its ID."`
	Export    LocationsExportCmd    `cmd:"" help:"Export locations to an Excel (.xlsx) file. Supports all the same filters as 'list'. The API generates the Excel file server-side with all resource fields included."`
	Delete    LocationsDeleteCmd    `cmd:"" help:"Delete a location by its ID."`
	Publish   LocationsPublishCmd   `cmd:"" help:"Publish a location, making it publicly visible."`
	Unpublish LocationsUnpublishCmd `cmd:"" help:"Unpublish a location, hiding it from public view."`
	Comments  LocationsCommentsCmd  `cmd:"" help:"List all comments on a location."`
	Comment   LocationsCommentCmd   `cmd:"" help:"Add a comment to a location."`
	Revisions LocationsRevisionsCmd `cmd:"" help:"Show the revision history of a location."`
}

type LocationsListCmd struct {
	Search       string `short:"s" help:"Full-text search query. Supports 'tag:keyword' and 'marker:name' syntax."`
	Markers      string `help:"Comma-separated markers filter. Prefix with '!' to exclude."`
	Keywords     string `help:"Comma-separated keywords filter."`
	Types        string `help:"Comma-separated category types filter."`
	Categories   string `help:"Comma-separated categories filter."`
	WFStatus     string `short:"w" enum:"draft,readyforvalidation,approved,rejected,deleted,archived," default:"" help:"Filter by workflow status."`
	Published    string `help:"Filter by published state (true/false)."`
	Deleted      bool   `help:"Include deleted items."`
	Owner        string `help:"Filter by owner."`
	UserOrg      string `name:"userorganisation" help:"Filter by user organisation."`
	TRCID        string `name:"trcid" help:"Filter by TRC ID."`
	ExternalID   string `name:"externalid" help:"Filter by external ID."`
	Language     string `name:"lang" help:"Filter by language (nl, en, de)."`
	UpdatedSince string `name:"updated-since" help:"Items updated after date. Relative: 2w, 3d, 1mo, 1y. Absolute: 2026-01-15."`
	Sort         string `short:"o" default:"modified" enum:"modified,created,title,wfstatus" help:"Sort field (default: modified)."`
	Asc          bool   `help:"Sort ascending (default: descending)."`
	Size         int    `short:"l" default:"25" help:"Results per page (default: 25, max: 5000)."`
	Page         int    `short:"p" default:"0" help:"Page number (0-indexed)."`
	JSON         bool   `short:"j" help:"Output as JSON."`
}

func (c *LocationsListCmd) Run(client *api.Client) error {
	opts := api.ListOptions{
		Search:     c.Search,
		Markers:    c.Markers,
		Keywords:   c.Keywords,
		Types:      c.Types,
		Categories: c.Categories,
		WFStatus:   c.WFStatus,
		Published:  c.Published,
		Deleted:    c.Deleted,
		Owner:      c.Owner,
		UserOrg:    c.UserOrg,
		TRCID:      c.TRCID,
		ExternalID: c.ExternalID,
		Language:   c.Language,
		Sort:       c.Sort,
		Asc:        c.Asc,
		Size:       c.Size,
		Page:       c.Page,
	}

	if c.UpdatedSince != "" {
		iso, err := ParseRelativeISO(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("--updated-since: %w", err)
		}
		opts.UpdatedSince = iso
	}

	result, err := client.ListLocations(opts)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(mustMarshal(result))
	}

	resources, err := api.ParseResources(result.Results)
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		fmt.Println("No locations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tCITY\tSTATUS\tPUBLISHED")
	fmt.Fprintln(w, "--\t-----\t----\t------\t---------")

	for _, r := range resources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.ID, truncate(r.GetTitle(), 40), truncate(r.GetCity(), 20), r.WFStatus, boolYesNo(r.Published))
	}
	w.Flush()

	fmt.Printf("\nShowing %d of %d locations (page %d)\n", len(resources), result.Hits, result.Page)
	return nil
}

type LocationsExportCmd struct {
	Output       string `short:"o" required:"" help:"Output file path for the Excel export (e.g. locations.xlsx)."`
	PropertyIDs  string `name:"export-propertyids" help:"Comma-separated list of category property IDs to include as additional columns in the Excel export. Each ID maps to a category property whose value is added as an extra column. Use 'tff dictionary categories' to find IDs."`
	Search       string `short:"s" help:"Full-text search query. Supports 'tag:keyword' and 'marker:name' syntax."`
	Markers      string `help:"Comma-separated markers filter. Prefix with '!' to exclude."`
	Keywords     string `help:"Comma-separated keywords filter."`
	Types        string `help:"Comma-separated category types filter."`
	Categories   string `help:"Comma-separated categories filter."`
	WFStatus     string `short:"w" enum:"draft,readyforvalidation,approved,rejected,deleted,archived," default:"" help:"Filter by workflow status."`
	Published    string `help:"Filter by published state (true/false)."`
	Deleted      bool   `help:"Include deleted items."`
	Owner        string `help:"Filter by owner."`
	UserOrg      string `name:"userorganisation" help:"Filter by user organisation."`
	TRCID        string `name:"trcid" help:"Filter by TRC ID."`
	ExternalID   string `name:"externalid" help:"Filter by external ID."`
	Language     string `name:"lang" help:"Filter by language (nl, en, de)."`
	UpdatedSince string `name:"updated-since" help:"Items updated after date. Relative: 2w, 3d, 1mo, 1y. Absolute: 2026-01-15."`
	Sort         string `enum:"modified,created,title,wfstatus," default:"" help:"Sort field."`
	Asc          bool   `help:"Sort ascending."`
}

func (c *LocationsExportCmd) Run(client *api.Client) error {
	opts := api.ListOptions{
		Search:     c.Search,
		Markers:    c.Markers,
		Keywords:   c.Keywords,
		Types:      c.Types,
		Categories: c.Categories,
		WFStatus:   c.WFStatus,
		Published:  c.Published,
		Deleted:    c.Deleted,
		Owner:      c.Owner,
		UserOrg:    c.UserOrg,
		TRCID:      c.TRCID,
		ExternalID: c.ExternalID,
		Language:   c.Language,
		Sort:       c.Sort,
		Asc:        c.Asc,
	}

	if c.UpdatedSince != "" {
		iso, err := ParseRelativeISO(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("--updated-since: %w", err)
		}
		opts.UpdatedSince = iso
	}

	exportOpts := api.ExportOptions{PropertyIDs: c.PropertyIDs}

	data, err := client.ExportLocations(opts, exportOpts)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.Output, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Printf("Exported locations to %s (%d bytes)\n", c.Output, len(data))
	return nil
}

type LocationsGetCmd struct {
	ID   string `arg:"" help:"Location ID."`
	JSON bool   `short:"j" help:"Output full JSON response."`
}

func (c *LocationsGetCmd) Run(client *api.Client) error {
	body, err := client.GetResource("locations", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	var r api.Resource
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("parsing location: %w", err)
	}

	printResourceDetail(r, "Location")
	return nil
}

type LocationsDeleteCmd struct {
	ID    string `arg:"" help:"Location ID to delete."`
	Force bool   `short:"f" help:"Skip confirmation prompt."`
}

func (c *LocationsDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete location %s? [y/N] ", c.ID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteResource("locations", c.ID); err != nil {
		return fmt.Errorf("deleting location: %w", err)
	}
	fmt.Printf("Location %s deleted.\n", c.ID)
	return nil
}

type LocationsPublishCmd struct {
	ID string `arg:"" help:"Location ID to publish."`
}

func (c *LocationsPublishCmd) Run(client *api.Client) error {
	if err := client.PublishResource("locations", c.ID); err != nil {
		return fmt.Errorf("publishing location: %w", err)
	}
	fmt.Printf("Location %s published.\n", c.ID)
	return nil
}

type LocationsUnpublishCmd struct {
	ID string `arg:"" help:"Location ID to unpublish."`
}

func (c *LocationsUnpublishCmd) Run(client *api.Client) error {
	if err := client.UnpublishResource("locations", c.ID); err != nil {
		return fmt.Errorf("unpublishing location: %w", err)
	}
	fmt.Printf("Location %s unpublished.\n", c.ID)
	return nil
}

type LocationsCommentsCmd struct {
	ID   string `arg:"" help:"Location ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *LocationsCommentsCmd) Run(client *api.Client) error {
	body, err := client.GetComments("locations", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printComments(body)
}

type LocationsCommentCmd struct {
	ID      string `arg:"" help:"Location ID."`
	Message string `arg:"" help:"Comment message."`
}

func (c *LocationsCommentCmd) Run(client *api.Client) error {
	if err := client.AddComment("locations", c.ID, c.Message); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	fmt.Printf("Comment added to location %s.\n", c.ID)
	return nil
}

type LocationsRevisionsCmd struct {
	ID   string `arg:"" help:"Location ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *LocationsRevisionsCmd) Run(client *api.Client) error {
	body, err := client.GetRevisions("locations", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printRevisions(body)
}
