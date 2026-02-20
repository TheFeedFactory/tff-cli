package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type EventGroupsCmd struct {
	List      EventGroupsListCmd      `cmd:"" help:"List and search event groups. Supports full-text search, workflow status filtering, markers, keywords, and more."`
	Get       EventGroupsGetCmd       `cmd:"" help:"Get detailed information about a specific event group by its ID."`
	Export    EventGroupsExportCmd    `cmd:"" help:"Export event groups to an Excel (.xlsx) file. Supports all list filters."`
	Delete    EventGroupsDeleteCmd    `cmd:"" help:"Delete an event group by its ID."`
	Publish   EventGroupsPublishCmd   `cmd:"" help:"Publish an event group, making it publicly visible."`
	Unpublish EventGroupsUnpublishCmd `cmd:"" help:"Unpublish an event group, hiding it from public view."`
	Comments  EventGroupsCommentsCmd  `cmd:"" help:"List all comments on an event group."`
	Comment   EventGroupsCommentCmd   `cmd:"" help:"Add a comment to an event group."`
	Revisions EventGroupsRevisionsCmd `cmd:"" help:"Show the revision history of an event group."`
}

type EventGroupsListCmd struct {
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

func (c *EventGroupsListCmd) Run(client *api.Client) error {
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

	result, err := client.ListEventGroups(opts)
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
		fmt.Println("No event groups found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tPUBLISHED")
	fmt.Fprintln(w, "--\t-----\t------\t---------")

	for _, r := range resources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			r.ID, truncate(r.GetTitle(), 50), r.WFStatus, boolYesNo(r.Published))
	}
	w.Flush()

	fmt.Printf("\nShowing %d of %d event groups (page %d)\n", len(resources), result.Hits, result.Page)
	return nil
}

type EventGroupsExportCmd struct {
	Output       string `short:"o" required:"" help:"Output file path (e.g. eventgroups.xlsx)."`
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

func (c *EventGroupsExportCmd) Run(client *api.Client) error {
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

	data, err := client.ExportEventGroups(opts)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.Output, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Printf("Exported event groups to %s (%d bytes)\n", c.Output, len(data))
	return nil
}

type EventGroupsGetCmd struct {
	ID   string `arg:"" help:"Event group ID."`
	JSON bool   `short:"j" help:"Output full JSON response."`
}

func (c *EventGroupsGetCmd) Run(client *api.Client) error {
	body, err := client.GetResource("eventgroups", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	var r api.Resource
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("parsing event group: %w", err)
	}

	printResourceDetail(r, "Event Group")
	return nil
}

type EventGroupsDeleteCmd struct {
	ID    string `arg:"" help:"Event group ID to delete."`
	Force bool   `short:"f" help:"Skip confirmation prompt."`
}

func (c *EventGroupsDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete event group %s? [y/N] ", c.ID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteResource("eventgroups", c.ID); err != nil {
		return fmt.Errorf("deleting event group: %w", err)
	}
	fmt.Printf("Event group %s deleted.\n", c.ID)
	return nil
}

type EventGroupsPublishCmd struct {
	ID string `arg:"" help:"Event group ID to publish."`
}

func (c *EventGroupsPublishCmd) Run(client *api.Client) error {
	if err := client.PublishResource("eventgroups", c.ID); err != nil {
		return fmt.Errorf("publishing event group: %w", err)
	}
	fmt.Printf("Event group %s published.\n", c.ID)
	return nil
}

type EventGroupsUnpublishCmd struct {
	ID string `arg:"" help:"Event group ID to unpublish."`
}

func (c *EventGroupsUnpublishCmd) Run(client *api.Client) error {
	if err := client.UnpublishResource("eventgroups", c.ID); err != nil {
		return fmt.Errorf("unpublishing event group: %w", err)
	}
	fmt.Printf("Event group %s unpublished.\n", c.ID)
	return nil
}

type EventGroupsCommentsCmd struct {
	ID   string `arg:"" help:"Event group ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *EventGroupsCommentsCmd) Run(client *api.Client) error {
	body, err := client.GetComments("eventgroups", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printComments(body)
}

type EventGroupsCommentCmd struct {
	ID      string `arg:"" help:"Event group ID."`
	Message string `arg:"" help:"Comment message."`
}

func (c *EventGroupsCommentCmd) Run(client *api.Client) error {
	if err := client.AddComment("eventgroups", c.ID, c.Message); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	fmt.Printf("Comment added to event group %s.\n", c.ID)
	return nil
}

type EventGroupsRevisionsCmd struct {
	ID   string `arg:"" help:"Event group ID."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *EventGroupsRevisionsCmd) Run(client *api.Client) error {
	body, err := client.GetRevisions("eventgroups", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printRevisions(body)
}
