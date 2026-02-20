package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type VenuesCmd struct {
	List      VenuesListCmd      `cmd:"" help:"List and search venues. Supports full-text search, workflow status filtering, markers, keywords, and more."`
	Get       VenuesGetCmd       `cmd:"" help:"Get detailed information about a specific venue by its ID."`
	Export    VenuesExportCmd    `cmd:"" help:"Export venues to an Excel (.xlsx) file. Supports all list filters plus --export-propertyids for custom category property columns."`
	Delete    VenuesDeleteCmd    `cmd:"" help:"Delete a venue by its ID."`
	Publish   VenuesPublishCmd   `cmd:"" help:"Publish a venue, making it publicly visible."`
	Unpublish VenuesUnpublishCmd `cmd:"" help:"Unpublish a venue, hiding it from public view."`
	Comments  VenuesCommentsCmd  `cmd:"" help:"List all comments on a venue."`
	Comment   VenuesCommentCmd   `cmd:"" help:"Add a comment to a venue."`
	Revisions VenuesRevisionsCmd `cmd:"" help:"Show the revision history of a venue."`
}

type VenuesListCmd struct {
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

func (c *VenuesListCmd) Run(client *api.Client) error {
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

	result, err := client.ListVenues(opts)
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
		fmt.Println("No venues found.")
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

	fmt.Printf("\nShowing %d of %d venues (page %d)\n", len(resources), result.Hits, result.Page)
	return nil
}

type VenuesExportCmd struct {
	Output       string `short:"o" required:"" help:"Output file path (e.g. venues.xlsx)."`
	PropertyIDs  string `name:"export-propertyids" help:"Comma-separated category property IDs for additional Excel columns. Use 'tff dictionary categories' to find IDs."`
	Search       string `short:"s" help:"Full-text search query."`
	Markers      string `help:"Comma-separated markers filter."`
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
	UpdatedSince string `name:"updated-since" help:"Items updated after date."`
	Sort         string `enum:"modified,created,title,wfstatus," default:"" help:"Sort field."`
	Asc          bool   `help:"Sort ascending."`
}

func (c *VenuesExportCmd) Run(client *api.Client) error {
	opts := api.ListOptions{
		Search: c.Search, Markers: c.Markers, Keywords: c.Keywords,
		Types: c.Types, Categories: c.Categories, WFStatus: c.WFStatus,
		Published: c.Published, Deleted: c.Deleted, Owner: c.Owner,
		UserOrg: c.UserOrg, TRCID: c.TRCID, ExternalID: c.ExternalID,
		Language: c.Language, Sort: c.Sort, Asc: c.Asc,
	}
	if c.UpdatedSince != "" {
		iso, err := ParseRelativeISO(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("--updated-since: %w", err)
		}
		opts.UpdatedSince = iso
	}

	data, err := client.ExportVenues(opts, api.ExportOptions{PropertyIDs: c.PropertyIDs})
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.Output, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Printf("Exported venues to %s (%d bytes)\n", c.Output, len(data))
	return nil
}

type VenuesGetCmd struct {
	ID   string `arg:"" help:"Venue ID."`
	JSON bool   `short:"j" help:"Output full JSON response."`
}

func (c *VenuesGetCmd) Run(client *api.Client) error {
	body, err := client.GetResource("venues", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	var r api.Resource
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("parsing venue: %w", err)
	}

	printResourceDetail(r, "Venue")
	return nil
}

type VenuesDeleteCmd struct {
	ID    string `arg:"" help:"Venue ID to delete."`
	Force bool   `short:"f" help:"Skip confirmation prompt."`
}

func (c *VenuesDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete venue %s? [y/N] ", c.ID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteResource("venues", c.ID); err != nil {
		return fmt.Errorf("deleting venue: %w", err)
	}
	fmt.Printf("Venue %s deleted.\n", c.ID)
	return nil
}

type VenuesPublishCmd struct {
	ID string `arg:"" help:"Venue ID to publish."`
}

func (c *VenuesPublishCmd) Run(client *api.Client) error {
	if err := client.PublishResource("venues", c.ID); err != nil {
		return fmt.Errorf("publishing venue: %w", err)
	}
	fmt.Printf("Venue %s published.\n", c.ID)
	return nil
}

type VenuesUnpublishCmd struct {
	ID string `arg:"" help:"Venue ID to unpublish."`
}

func (c *VenuesUnpublishCmd) Run(client *api.Client) error {
	if err := client.UnpublishResource("venues", c.ID); err != nil {
		return fmt.Errorf("unpublishing venue: %w", err)
	}
	fmt.Printf("Venue %s unpublished.\n", c.ID)
	return nil
}

type VenuesCommentsCmd struct {
	ID   string `arg:"" help:"Venue ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *VenuesCommentsCmd) Run(client *api.Client) error {
	body, err := client.GetComments("venues", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printComments(body)
}

type VenuesCommentCmd struct {
	ID      string `arg:"" help:"Venue ID."`
	Message string `arg:"" help:"Comment message."`
}

func (c *VenuesCommentCmd) Run(client *api.Client) error {
	if err := client.AddComment("venues", c.ID, c.Message); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	fmt.Printf("Comment added to venue %s.\n", c.ID)
	return nil
}

type VenuesRevisionsCmd struct {
	ID   string `arg:"" help:"Venue ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *VenuesRevisionsCmd) Run(client *api.Client) error {
	body, err := client.GetRevisions("venues", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printRevisions(body)
}
