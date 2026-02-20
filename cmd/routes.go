package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type RoutesCmd struct {
	List      RoutesListCmd      `cmd:"" help:"List and search routes. Supports full-text search, workflow status filtering, markers, keywords, and more."`
	Get       RoutesGetCmd       `cmd:"" help:"Get detailed information about a specific route by its ID."`
	Export    RoutesExportCmd    `cmd:"" help:"Export routes to an Excel (.xlsx) file. Supports all list filters."`
	Delete    RoutesDeleteCmd    `cmd:"" help:"Delete a route by its ID."`
	Publish   RoutesPublishCmd   `cmd:"" help:"Publish a route, making it publicly visible."`
	Unpublish RoutesUnpublishCmd `cmd:"" help:"Unpublish a route, hiding it from public view."`
	Comments  RoutesCommentsCmd  `cmd:"" help:"List all comments on a route."`
	Comment   RoutesCommentCmd   `cmd:"" help:"Add a comment to a route."`
	Revisions RoutesRevisionsCmd `cmd:"" help:"Show the revision history of a route."`
}

type RoutesListCmd struct {
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

func (c *RoutesListCmd) Run(client *api.Client) error {
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

	result, err := client.ListRoutes(opts)
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
		fmt.Println("No routes found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tTYPE\tDISTANCE\tSTATUS\tPUBLISHED")
	fmt.Fprintln(w, "--\t-----\t----\t--------\t------\t---------")

	for _, r := range resources {
		routeType := ""
		distance := ""
		if r.Physical != nil {
			routeType = r.Physical.RouteType
			distance = r.Physical.Distance
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, truncate(r.GetTitle(), 40), routeType, distance, r.WFStatus, boolYesNo(r.Published))
	}
	w.Flush()

	fmt.Printf("\nShowing %d of %d routes (page %d)\n", len(resources), result.Hits, result.Page)
	return nil
}

type RoutesExportCmd struct {
	Output       string `short:"o" required:"" help:"Output file path (e.g. routes.xlsx)."`
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

func (c *RoutesExportCmd) Run(client *api.Client) error {
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

	data, err := client.ExportRoutes(opts)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.Output, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Printf("Exported routes to %s (%d bytes)\n", c.Output, len(data))
	return nil
}

type RoutesGetCmd struct {
	ID   string `arg:"" help:"Route ID."`
	JSON bool   `short:"j" help:"Output full JSON response."`
}

func (c *RoutesGetCmd) Run(client *api.Client) error {
	body, err := client.GetResource("routes", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	var r api.Resource
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("parsing route: %w", err)
	}

	printResourceDetail(r, "Route")
	return nil
}

type RoutesDeleteCmd struct {
	ID    string `arg:"" help:"Route ID to delete."`
	Force bool   `short:"f" help:"Skip confirmation prompt."`
}

func (c *RoutesDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete route %s? [y/N] ", c.ID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteResource("routes", c.ID); err != nil {
		return fmt.Errorf("deleting route: %w", err)
	}
	fmt.Printf("Route %s deleted.\n", c.ID)
	return nil
}

type RoutesPublishCmd struct {
	ID string `arg:"" help:"Route ID to publish."`
}

func (c *RoutesPublishCmd) Run(client *api.Client) error {
	if err := client.PublishResource("routes", c.ID); err != nil {
		return fmt.Errorf("publishing route: %w", err)
	}
	fmt.Printf("Route %s published.\n", c.ID)
	return nil
}

type RoutesUnpublishCmd struct {
	ID string `arg:"" help:"Route ID to unpublish."`
}

func (c *RoutesUnpublishCmd) Run(client *api.Client) error {
	if err := client.UnpublishResource("routes", c.ID); err != nil {
		return fmt.Errorf("unpublishing route: %w", err)
	}
	fmt.Printf("Route %s unpublished.\n", c.ID)
	return nil
}

type RoutesCommentsCmd struct {
	ID   string `arg:"" help:"Route ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *RoutesCommentsCmd) Run(client *api.Client) error {
	body, err := client.GetComments("routes", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printComments(body)
}

type RoutesCommentCmd struct {
	ID      string `arg:"" help:"Route ID."`
	Message string `arg:"" help:"Comment message."`
}

func (c *RoutesCommentCmd) Run(client *api.Client) error {
	if err := client.AddComment("routes", c.ID, c.Message); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	fmt.Printf("Comment added to route %s.\n", c.ID)
	return nil
}

type RoutesRevisionsCmd struct {
	ID   string `arg:"" help:"Route ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *RoutesRevisionsCmd) Run(client *api.Client) error {
	body, err := client.GetRevisions("routes", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printRevisions(body)
}
