package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type EventsCmd struct {
	List      EventsListCmd      `cmd:"" help:"List and search events. Supports full-text search, date range filtering, geographic filtering, workflow status, markers, keywords, and more. Returns paginated results sorted by last modified date by default."`
	Get       EventsGetCmd       `cmd:"" help:"Get detailed information about a specific event by its ID. Returns all fields including title, description, calendar, location, media, and metadata."`
	Export    EventsExportCmd    `cmd:"" help:"Export events to an Excel (.xlsx) file. Supports all the same filters as 'list'. The API generates the Excel file server-side with all resource fields included."`
	Delete    EventsDeleteCmd    `cmd:"" help:"Delete an event by its ID. This sets the event's workflow status to deleted."`
	Publish   EventsPublishCmd   `cmd:"" help:"Publish an event, making it publicly visible. Sets the published flag to true."`
	Unpublish EventsUnpublishCmd `cmd:"" help:"Unpublish an event, hiding it from public view. Sets the published flag to false."`
	Comments  EventsCommentsCmd  `cmd:"" help:"List all comments on an event. Comments are internal notes visible to editors."`
	Comment   EventsCommentCmd   `cmd:"" help:"Add a comment to an event. Comments are internal notes visible to editors."`
	Revisions EventsRevisionsCmd `cmd:"" help:"Show the revision history of an event, including who made changes and when."`
}

type EventsListCmd struct {
	Search       string `short:"s" help:"Full-text search query. Searches across title, description, and other text fields. Supports special syntax: 'tag:keyword' to search by keyword tag, 'marker:name' to search by marker name."`
	Markers      string `help:"Comma-separated list of markers to filter by. Prefix with '!' to exclude a marker. Example: '!marker1,marker2' excludes marker1 but requires marker2."`
	Keywords     string `help:"Comma-separated list of keywords to filter by."`
	Types        string `help:"Comma-separated category types to filter by."`
	Categories   string `help:"Comma-separated categories to filter by."`
	WFStatus     string `short:"w" enum:"draft,readyforvalidation,approved,rejected,deleted,archived," default:"" help:"Filter by workflow status. Allowed values: draft, readyforvalidation, approved, rejected, deleted, archived."`
	Published    string `help:"Filter by published state. Use 'true' for published events, 'false' for unpublished."`
	Deleted      bool   `help:"Include deleted events in results. Default: false."`
	Owner        string `help:"Filter by owner (username or email)."`
	UserOrg      string `name:"userorganisation" help:"Filter by user organisation. Use the account name without spaces."`
	TRCID        string `name:"trcid" help:"Filter by TRC ID (Toeristische Recreatieve Content identifier)."`
	ExternalID   string `name:"externalid" help:"Filter by external ID."`
	Language     string `name:"lang" help:"Filter by language. Supported: nl, en, de."`
	UpdatedSince string `name:"updated-since" help:"Show events updated after this date. Supports relative time: 2w (2 weeks ago), 3d (3 days ago), 1mo (1 month ago), 1y (1 year ago). Also supports absolute dates: 2026-01-15."`
	Sort         string `short:"o" default:"modified" enum:"modified,created,title,wfstatus" help:"Sort results by field. Options: modified (default), created, title, wfstatus."`
	Asc          bool   `help:"Sort in ascending order. Default is descending (newest first)."`
	Size         int    `short:"l" default:"25" help:"Number of results per page. Default: 25, maximum: 5000."`
	Page         int    `short:"p" default:"0" help:"Page number (0-indexed). Default: 0."`
	JSON         bool   `short:"j" help:"Output full API response as JSON instead of a table."`

	// Event-specific flags
	DateFrom    string `name:"date-from" help:"Filter events starting from this date. Supports relative time (1w, 2mo) or absolute date (yyyy-mm-dd)."`
	DateTo      string `name:"date-to" help:"Filter events up to this date. Supports relative time (1w, 2mo) or absolute date (yyyy-mm-dd)."`
	LocationID  string `name:"location-id" help:"Filter events by location ID."`
	City        string `help:"Filter events by city name."`
	Geo         string `help:"Geographic center point for distance filtering. Format: lat,lon (e.g. 52.37,4.89). Use with --geo-distance."`
	GeoDistance string `name:"geo-distance" help:"Maximum distance from --geo point. Format: number followed by unit (e.g. 10km, 5mi). Requires --geo flag."`
}

func (c *EventsListCmd) Run(client *api.Client) error {
	opts := api.EventListOptions{
		ListOptions: api.ListOptions{
			Search:   c.Search,
			Markers:  c.Markers,
			Keywords: c.Keywords,
			Types:    c.Types,
			Categories: c.Categories,
			WFStatus: c.WFStatus,
			Published: c.Published,
			Deleted:  c.Deleted,
			Owner:    c.Owner,
			UserOrg:  c.UserOrg,
			TRCID:    c.TRCID,
			ExternalID: c.ExternalID,
			Language: c.Language,
			Sort:     c.Sort,
			Asc:      c.Asc,
			Size:     c.Size,
			Page:     c.Page,
		},
		LocationID: c.LocationID,
		City:       c.City,
	}

	// Parse updated-since
	if c.UpdatedSince != "" {
		iso, err := ParseRelativeISO(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("--updated-since: %w", err)
		}
		opts.ListOptions.UpdatedSince = iso
	}

	// Parse date-from
	if c.DateFrom != "" {
		d, err := ParseRelativeDate(c.DateFrom)
		if err != nil {
			return fmt.Errorf("--date-from: %w", err)
		}
		opts.DateFrom = d
	}

	// Parse date-to
	if c.DateTo != "" {
		d, err := ParseRelativeDate(c.DateTo)
		if err != nil {
			return fmt.Errorf("--date-to: %w", err)
		}
		opts.DateTo = d
	}

	// Parse geo
	if c.Geo != "" {
		parts := strings.SplitN(c.Geo, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--geo must be in format lat,lon (e.g. 52.37,4.89)")
		}
		opts.GeoLat = strings.TrimSpace(parts[0])
		opts.GeoLon = strings.TrimSpace(parts[1])
	}
	if c.GeoDistance != "" {
		if c.Geo == "" {
			return fmt.Errorf("--geo-distance requires --geo flag")
		}
		opts.GeoDistance = c.GeoDistance
	}

	result, err := client.ListEvents(opts)
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
		fmt.Println("No events found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tCITY\tDATE\tSTATUS\tPUBLISHED")
	fmt.Fprintln(w, "--\t-----\t----\t----\t------\t---------")

	for _, r := range resources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID,
			truncate(r.GetTitle(), 40),
			truncate(r.GetCity(), 20),
			r.GetFirstDate(),
			r.WFStatus,
			boolYesNo(r.Published),
		)
	}
	w.Flush()

	fmt.Printf("\nShowing %d of %d events (page %d)\n", len(resources), result.Hits, result.Page)
	return nil
}

type EventsExportCmd struct {
	Output       string `short:"o" required:"" help:"Output file path (e.g. events.xlsx)."`
	Format       string `enum:"excel,uitkrant," default:"excel" help:"Export format. 'excel' for Excel spreadsheet (.xlsx), 'uitkrant' for plain text publication format (requires --date-from and --date-to)."`
	PropertyIDs  string `name:"export-propertyids" help:"Comma-separated list of category property IDs to include as additional columns in the Excel export. Each ID maps to a category property whose value is added as an extra column. Use 'tff dictionary categories' to find IDs."`
	Search       string `short:"s" help:"Full-text search query. Supports 'tag:keyword' and 'marker:name' syntax."`
	Markers      string `help:"Comma-separated list of markers to filter by. Prefix with '!' to exclude."`
	Keywords     string `help:"Comma-separated list of keywords to filter by."`
	Types        string `help:"Comma-separated category types to filter by."`
	Categories   string `help:"Comma-separated categories to filter by."`
	WFStatus     string `short:"w" enum:"draft,readyforvalidation,approved,rejected,deleted,archived," default:"" help:"Filter by workflow status."`
	Published    string `help:"Filter by published state (true/false)."`
	Deleted      bool   `help:"Include deleted events."`
	Owner        string `help:"Filter by owner."`
	UserOrg      string `name:"userorganisation" help:"Filter by user organisation."`
	TRCID        string `name:"trcid" help:"Filter by TRC ID."`
	ExternalID   string `name:"externalid" help:"Filter by external ID."`
	Language     string `name:"lang" help:"Filter by language (nl, en, de)."`
	UpdatedSince string `name:"updated-since" help:"Items updated after date. Relative: 2w, 3d, 1mo, 1y. Absolute: 2026-01-15."`
	Sort         string `enum:"modified,created,title,wfstatus," default:"" help:"Sort field."`
	Asc          bool   `help:"Sort ascending."`
	DateFrom     string `name:"date-from" help:"Event date range start (yyyy-mm-dd or relative)."`
	DateTo       string `name:"date-to" help:"Event date range end (yyyy-mm-dd or relative)."`
	LocationID   string `name:"location-id" help:"Filter by location ID."`
	City         string `help:"Filter by city name."`
	Geo          string `help:"Geographic filter as lat,lon (e.g. 52.37,4.89)."`
	GeoDistance  string `name:"geo-distance" help:"Distance for geo filter (e.g. 10km). Requires --geo."`
}

func (c *EventsExportCmd) Run(client *api.Client) error {
	if c.Format == "uitkrant" && (c.DateFrom == "" || c.DateTo == "") {
		return fmt.Errorf("format 'uitkrant' requires both --date-from and --date-to")
	}

	opts := api.EventListOptions{
		ListOptions: api.ListOptions{
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
		},
		LocationID: c.LocationID,
		City:       c.City,
	}

	exportOpts := api.ExportOptions{
		PropertyIDs: c.PropertyIDs,
		Format:      c.Format,
	}

	if c.UpdatedSince != "" {
		iso, err := ParseRelativeISO(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("--updated-since: %w", err)
		}
		opts.ListOptions.UpdatedSince = iso
	}
	if c.DateFrom != "" {
		d, err := ParseRelativeDate(c.DateFrom)
		if err != nil {
			return fmt.Errorf("--date-from: %w", err)
		}
		opts.DateFrom = d
	}
	if c.DateTo != "" {
		d, err := ParseRelativeDate(c.DateTo)
		if err != nil {
			return fmt.Errorf("--date-to: %w", err)
		}
		opts.DateTo = d
	}
	if c.Geo != "" {
		parts := strings.SplitN(c.Geo, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--geo must be in format lat,lon (e.g. 52.37,4.89)")
		}
		opts.GeoLat = strings.TrimSpace(parts[0])
		opts.GeoLon = strings.TrimSpace(parts[1])
	}
	if c.GeoDistance != "" {
		if c.Geo == "" {
			return fmt.Errorf("--geo-distance requires --geo flag")
		}
		opts.GeoDistance = c.GeoDistance
	}

	data, err := client.ExportEvents(opts, exportOpts)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.Output, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Printf("Exported events to %s (%d bytes)\n", c.Output, len(data))
	return nil
}

type EventsGetCmd struct {
	ID   string `arg:"" help:"Event ID (required)."`
	JSON bool   `short:"j" help:"Output full JSON response instead of formatted text."`
}

func (c *EventsGetCmd) Run(client *api.Client) error {
	body, err := client.GetResource("events", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	var r api.Resource
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}

	printResourceDetail(r, "Event")
	return nil
}

type EventsDeleteCmd struct {
	ID    string `arg:"" help:"Event ID to delete."`
	Force bool   `short:"f" help:"Skip confirmation prompt."`
}

func (c *EventsDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete event %s? [y/N] ", c.ID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteResource("events", c.ID); err != nil {
		return fmt.Errorf("deleting event: %w", err)
	}
	fmt.Printf("Event %s deleted.\n", c.ID)
	return nil
}

type EventsPublishCmd struct {
	ID string `arg:"" help:"Event ID to publish."`
}

func (c *EventsPublishCmd) Run(client *api.Client) error {
	if err := client.PublishResource("events", c.ID); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}
	fmt.Printf("Event %s published.\n", c.ID)
	return nil
}

type EventsUnpublishCmd struct {
	ID string `arg:"" help:"Event ID to unpublish."`
}

func (c *EventsUnpublishCmd) Run(client *api.Client) error {
	if err := client.UnpublishResource("events", c.ID); err != nil {
		return fmt.Errorf("unpublishing event: %w", err)
	}
	fmt.Printf("Event %s unpublished.\n", c.ID)
	return nil
}

type EventsCommentsCmd struct {
	ID   string `arg:"" help:"Event ID to list comments for."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *EventsCommentsCmd) Run(client *api.Client) error {
	body, err := client.GetComments("events", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printComments(body)
}

type EventsCommentCmd struct {
	ID      string `arg:"" help:"Event ID to comment on."`
	Message string `arg:"" help:"Comment message text."`
}

func (c *EventsCommentCmd) Run(client *api.Client) error {
	if err := client.AddComment("events", c.ID, c.Message); err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	fmt.Printf("Comment added to event %s.\n", c.ID)
	return nil
}

type EventsRevisionsCmd struct {
	ID   string `arg:"" help:"Event ID to show revisions for."`
	JSON bool   `short:"j" help:"Output as JSON."`
}

func (c *EventsRevisionsCmd) Run(client *api.Client) error {
	body, err := client.GetRevisions("events", c.ID)
	if err != nil {
		return err
	}

	if c.JSON {
		return printRawJSON(body)
	}

	return printRevisions(body)
}

// Shared helper functions used by all resource commands

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func printResourceDetail(r api.Resource, resourceType string) {
	fmt.Printf("%s: %s\n", resourceType, r.GetTitle())
	fmt.Printf("ID: %s\n", r.ID)
	if r.Slug != "" {
		fmt.Printf("Slug: %s\n", r.Slug)
	}
	if r.TRCID != "" {
		fmt.Printf("TRC ID: %s\n", r.TRCID)
	}
	if r.ExternalID != "" {
		fmt.Printf("External ID: %s\n", r.ExternalID)
	}
	fmt.Printf("Status: %s\n", r.WFStatus)
	fmt.Printf("Published: %s\n", boolYesNo(r.Published))
	if r.Deleted {
		fmt.Printf("Deleted: Yes\n")
	}
	if r.Owner != "" {
		fmt.Printf("Owner: %s\n", r.Owner)
	}
	if r.UserOrg != "" {
		fmt.Printf("Organisation: %s\n", r.UserOrg)
	}
	if r.EntityType != "" {
		fmt.Printf("Type: %s\n", r.EntityType)
	}

	// Titles and descriptions in all languages
	if len(r.TRCItemDetails) > 0 {
		if len(r.TRCItemDetails) > 1 {
			fmt.Println("\nTitles:")
			for _, d := range r.TRCItemDetails {
				fmt.Printf("  %s: %s\n", d.Lang, d.Title)
			}
		}

		fmt.Println("\nShort Description:")
		for _, d := range r.TRCItemDetails {
			if d.ShortDescription != "" {
				fmt.Printf("  %s: %s\n", d.Lang, truncate(d.ShortDescription, 200))
			}
		}
	}

	// Location
	if r.Location != nil && r.Location.Address != nil {
		a := r.Location.Address
		fmt.Println("\nLocation:")
		if a.Street != "" {
			line := a.Street
			if a.HouseNr != "" {
				line += " " + a.HouseNr
			}
			fmt.Printf("  Address: %s\n", line)
		}
		if a.ZipCode != "" || a.City != "" {
			fmt.Printf("  City: %s %s\n", a.ZipCode, a.City)
		}
		if a.Latitude != 0 || a.Longitude != 0 {
			fmt.Printf("  Coordinates: %.6f, %.6f\n", a.Latitude, a.Longitude)
		}
	}

	// Calendar (events)
	if r.Calendar != nil && len(r.Calendar.SingleDates) > 0 {
		fmt.Println("\nDates:")
		limit := len(r.Calendar.SingleDates)
		if limit > 10 {
			limit = 10
		}
		for _, d := range r.Calendar.SingleDates[:limit] {
			line := d.Date
			if d.StartTime != "" {
				line += " " + d.StartTime
			}
			if d.EndTime != "" {
				line += " - " + d.EndTime
			}
			fmt.Printf("  %s\n", line)
		}
		if len(r.Calendar.SingleDates) > 10 {
			fmt.Printf("  ... and %d more dates\n", len(r.Calendar.SingleDates)-10)
		}
	}

	// Physical (routes)
	if r.Physical != nil {
		if r.Physical.RouteType != "" {
			fmt.Printf("\nRoute Type: %s\n", r.Physical.RouteType)
		}
		if r.Physical.Distance != "" {
			fmt.Printf("Distance: %s\n", r.Physical.Distance)
		}
		if r.Physical.Duration != "" {
			fmt.Printf("Duration: %s\n", r.Physical.Duration)
		}
	}

	// Contact
	if r.ContactInfo != nil {
		phone := r.ContactInfo.GetPhone()
		email := r.ContactInfo.GetEmail()
		if phone != "" || email != "" {
			fmt.Println("\nContact:")
			if phone != "" {
				fmt.Printf("  Phone: %s\n", phone)
			}
			if email != "" {
				fmt.Printf("  Email: %s\n", email)
			}
		}
		if len(r.ContactInfo.URLs) > 0 {
			fmt.Println("\nContact URLs:")
			for _, u := range r.ContactInfo.URLs {
				label := u.URLServiceType
				if label == "" {
					label = "url"
				}
				fmt.Printf("  %s: %s\n", label, u.URL)
			}
		}
	}

	// URLs
	if len(r.URLs) > 0 {
		fmt.Println("\nURLs:")
		for _, u := range r.URLs {
			label := u.URLType
			if u.Label != "" {
				label = u.Label
			}
			fmt.Printf("  %s: %s\n", label, u.URL)
		}
	}

	// Media
	if len(r.Media) > 0 {
		fmt.Println("\nMedia:")
		for _, m := range r.Media {
			main := ""
			if m.Main {
				main = " (main)"
			}
			fmt.Printf("  %s%s: %s\n", m.MediaType, main, m.URL)
		}
	}

	// Types / Categories
	if len(r.Types) > 0 {
		fmt.Printf("\nTypes: %s\n", strings.Join(r.Types, ", "))
	}

	// Keywords
	keywords := r.GetKeywords()
	if len(keywords) > 0 {
		fmt.Println("\nKeywords:")
		for _, k := range keywords {
			label := k.Label
			if label == "" {
				label = k.Value
			}
			fmt.Printf("  %s\n", label)
		}
	}

	// Markers
	markers := r.GetMarkers()
	if len(markers) > 0 {
		fmt.Printf("\nMarkers: %s\n", strings.Join(markers, ", "))
	}

	// Dates
	fmt.Println()
	if r.Created != "" {
		fmt.Printf("Created: %s\n", r.Created)
	}
	if r.LastUpdated != "" {
		fmt.Printf("Last Updated: %s\n", r.LastUpdated)
	}
}

func printComments(body []byte) error {
	var comments []api.Comment
	if err := json.Unmarshal(body, &comments); err != nil {
		// Try as a wrapper object
		var wrapper struct {
			Comments []api.Comment `json:"comments"`
		}
		if err2 := json.Unmarshal(body, &wrapper); err2 != nil {
			// Might be empty or unexpected format, just print raw
			return printRawJSON(body)
		}
		comments = wrapper.Comments
	}

	if len(comments) == 0 {
		fmt.Println("No comments.")
		return nil
	}

	for _, c := range comments {
		fmt.Printf("[%s] %s:\n  %s\n\n", c.Created, c.Author, c.Text)
	}
	return nil
}

func printRevisions(body []byte) error {
	var revisions []api.Revision
	if err := json.Unmarshal(body, &revisions); err != nil {
		var wrapper struct {
			Revisions []api.Revision `json:"revisions"`
		}
		if err2 := json.Unmarshal(body, &wrapper); err2 != nil {
			return printRawJSON(body)
		}
		revisions = wrapper.Revisions
	}

	if len(revisions) == 0 {
		fmt.Println("No revisions.")
		return nil
	}

	for _, r := range revisions {
		comment := ""
		if r.Comment != "" {
			comment = " — " + r.Comment
		}
		fmt.Printf("[%s] %s%s\n", r.Created, r.Author, comment)
	}
	return nil
}
