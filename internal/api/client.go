package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/TheFeedFactory/tff-cli/internal/config"
)

const baseURL = "https://app.thefeedfactory.nl/api"

type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      cfg.Token,
	}
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	reqURL := baseURL + endpoint

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errMsg string
		var errResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			if msg, ok := errResp["message"].(string); ok && msg != "" {
				errMsg = msg
			} else if msg, ok := errResp["error"].(string); ok && msg != "" {
				errMsg = msg
			}
		}
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errMsg)
	}

	return respBody, nil
}

// SearchResult represents the paginated response from list endpoints.
type SearchResult struct {
	Size    int               `json:"size"`
	Page    int               `json:"page"`
	Hits    int               `json:"hits"`
	Results []json.RawMessage `json:"results"`
}

// Resource represents a generic FeedFactory resource (event, location, route, venue, eventgroup).
// The API returns titles and descriptions inside trcItemDetails, grouped by language.
type Resource struct {
	ID              string           `json:"id"`
	Slug            string           `json:"slug,omitempty"`
	Types           []string         `json:"types,omitempty"`
	Published       bool             `json:"published"`
	Offline         bool             `json:"offline,omitempty"`
	Deleted         bool             `json:"deleted,omitempty"`
	WFStatus        string           `json:"wfstatus"`
	LastUpdated     string           `json:"lastupdated"`
	LastUpdatedBy   string           `json:"lastupdatedby,omitempty"`
	Created         string           `json:"creationdate"`
	Owner           string           `json:"owner,omitempty"`
	Calendar        *Calendar        `json:"calendar,omitempty"`
	Location        *Location        `json:"location,omitempty"`
	Physical        *Physical        `json:"physical,omitempty"`
	ContactInfo     *ContactInfo     `json:"contactinfo,omitempty"`
	Media           []Media          `json:"media,omitempty"`
	URLs            []URLEntry       `json:"urls,omitempty"`
	Files           []interface{}    `json:"files,omitempty"`
	Markers         FlexStringSlice  `json:"markers,omitempty"`
	Keywords        json.RawMessage  `json:"keywords,omitempty"`
	UserOrg         string           `json:"userorganisation,omitempty"`
	TRCID           string           `json:"trcid,omitempty"`
	ExternalID      string           `json:"externalid,omitempty"`
	EntityType      string           `json:"entitytype,omitempty"`
	TRCItemDetails  []TRCItemDetail  `json:"trcItemDetails,omitempty"`
	Translations    *Translations    `json:"translations,omitempty"`
	Performers      []interface{}    `json:"performers,omitempty"`
	PriceElements   []interface{}    `json:"priceElements,omitempty"`
}

// FlexStringSlice handles JSON fields that can be either a string or an array of strings.
type FlexStringSlice []string

func (f *FlexStringSlice) UnmarshalJSON(data []byte) error {
	// Try as array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}
	// Try as single string (possibly comma-separated)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			*f = nil
		} else {
			*f = strings.Split(s, ",")
		}
		return nil
	}
	// Null or unparseable
	*f = nil
	return nil
}

// GetMarkers returns the markers as a string slice.
func (r *Resource) GetMarkers() []string {
	return []string(r.Markers)
}

// GetKeywords returns parsed keywords, handling both array and null.
func (r *Resource) GetKeywords() []Keyword {
	if r.Keywords == nil {
		return nil
	}
	var kws []Keyword
	if json.Unmarshal(r.Keywords, &kws) == nil {
		return kws
	}
	return nil
}

// TRCItemDetail holds language-specific content for a resource.
type TRCItemDetail struct {
	Lang             string `json:"lang"`
	Title            string `json:"title"`
	ShortDescription string `json:"shortdescription"`
	LongDescription  string `json:"longdescription"`
}

type Translations struct {
	AvailableLanguages []string `json:"availableLanguages,omitempty"`
	PrimaryLanguage    string   `json:"primaryLanguage,omitempty"`
}

// GetTitle returns the best available title, preferring nl > en > de > first available.
func (r *Resource) GetTitle() string {
	if len(r.TRCItemDetails) == 0 {
		return "-"
	}
	// Try preferred languages in order
	for _, lang := range []string{"nl", "en", "de"} {
		for _, d := range r.TRCItemDetails {
			if d.Lang == lang && d.Title != "" {
				return d.Title
			}
		}
	}
	// Fallback to first available
	if r.TRCItemDetails[0].Title != "" {
		return r.TRCItemDetails[0].Title
	}
	return "-"
}

// GetShortDescription returns the best available short description.
func (r *Resource) GetShortDescription() string {
	if len(r.TRCItemDetails) == 0 {
		return ""
	}
	for _, lang := range []string{"nl", "en", "de"} {
		for _, d := range r.TRCItemDetails {
			if d.Lang == lang && d.ShortDescription != "" {
				return d.ShortDescription
			}
		}
	}
	return r.TRCItemDetails[0].ShortDescription
}

// GetCity returns the city from the location address, if available.
func (r *Resource) GetCity() string {
	if r.Location != nil && r.Location.Address != nil {
		return r.Location.Address.City
	}
	return ""
}

// GetFirstDate returns the first single date, if available.
func (r *Resource) GetFirstDate() string {
	if r.Calendar != nil && len(r.Calendar.SingleDates) > 0 {
		return r.Calendar.SingleDates[0].Date
	}
	return ""
}

type Calendar struct {
	CalendarType string       `json:"calendarType,omitempty"`
	SingleDates  []SingleDate `json:"singleDates,omitempty"`
	PatternDates []interface{} `json:"patternDates,omitempty"`
	Cancelled    bool         `json:"cancelled,omitempty"`
	SoldOut      bool         `json:"soldout,omitempty"`
}

type SingleDate struct {
	Date      string `json:"date,omitempty"`
	StartTime string `json:"starttime,omitempty"`
	EndTime   string `json:"endtime,omitempty"`
}

type Location struct {
	Address *Address `json:"address,omitempty"`
	Label   string   `json:"label,omitempty"`
}

type Address struct {
	Street     string  `json:"street,omitempty"`
	HouseNr    string  `json:"housenr,omitempty"`
	ZipCode    string  `json:"zipcode,omitempty"`
	City       string  `json:"city,omitempty"`
	Country    string  `json:"country,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
}

type Physical struct {
	Distance   string `json:"distance,omitempty"`
	Duration   string `json:"duration,omitempty"`
	RouteType  string `json:"routetype,omitempty"`
}

// ContactInfo uses flexible types since the API returns both simple and complex contact structures.
type ContactInfo struct {
	// Simple fields (may be string or object)
	Phone json.RawMessage `json:"phone,omitempty"`
	Mail  json.RawMessage `json:"mail,omitempty"`

	// Array fields
	Phones    []ContactPhone `json:"phones,omitempty"`
	Mails     []ContactMail  `json:"mails,omitempty"`
	Faxes     []interface{}  `json:"faxes,omitempty"`
	URLs      []ContactURL   `json:"urls,omitempty"`
	Addresses []interface{}  `json:"addresses,omitempty"`
}

type ContactPhone struct {
	Number string `json:"number"`
}

type ContactMail struct {
	Email string `json:"email"`
}

type ContactURL struct {
	URL            string `json:"url"`
	TargetLanguage string `json:"targetLanguage,omitempty"`
	URLServiceType string `json:"urlServiceType,omitempty"`
}

// GetPhone returns the primary phone number.
func (ci *ContactInfo) GetPhone() string {
	// Try phones array first
	if len(ci.Phones) > 0 && ci.Phones[0].Number != "" {
		return ci.Phones[0].Number
	}
	// Try phone object
	if ci.Phone != nil {
		var p ContactPhone
		if json.Unmarshal(ci.Phone, &p) == nil && p.Number != "" {
			return p.Number
		}
		// Try as plain string
		var s string
		if json.Unmarshal(ci.Phone, &s) == nil && s != "" {
			return s
		}
	}
	return ""
}

// GetEmail returns the primary email.
func (ci *ContactInfo) GetEmail() string {
	if len(ci.Mails) > 0 && ci.Mails[0].Email != "" {
		return ci.Mails[0].Email
	}
	if ci.Mail != nil {
		var m ContactMail
		if json.Unmarshal(ci.Mail, &m) == nil && m.Email != "" {
			return m.Email
		}
		var s string
		if json.Unmarshal(ci.Mail, &s) == nil && s != "" {
			return s
		}
	}
	return ""
}

type Media struct {
	URL     string `json:"url,omitempty"`
	Main    bool   `json:"main,omitempty"`
	MediaType string `json:"mediatype,omitempty"`
	Title   string `json:"title,omitempty"`
}

type URLEntry struct {
	URL       string `json:"url,omitempty"`
	URLType   string `json:"urltype,omitempty"`
	Label     string `json:"label,omitempty"`
}

type Keyword struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

type Comment struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Author    string `json:"author"`
	Created   string `json:"created"`
}

type Revision struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Created   string `json:"created"`
	Comment   string `json:"comment,omitempty"`
}

type Account struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

type DictionaryItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
}

type Organisation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListOptions contains common query parameters for list endpoints.
type ListOptions struct {
	Search       string
	Markers      string
	Keywords     string
	Types        string
	Categories   string
	WFStatus     string
	Published    string
	Deleted      bool
	Owner        string
	UserOrg      string
	TRCID        string
	ExternalID   string
	Language     string
	UpdatedSince string
	Sort         string
	Asc          bool
	Size         int
	Page         int
}

// EventListOptions extends ListOptions with event-specific parameters.
type EventListOptions struct {
	ListOptions
	DateFrom    string
	DateTo      string
	LocationID  string
	City        string
	GeoLat      string
	GeoLon      string
	GeoDistance  string
}

func buildListQuery(opts ListOptions) url.Values {
	q := url.Values{}

	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.Markers != "" {
		q.Set("markers", opts.Markers)
	}
	if opts.Keywords != "" {
		q.Set("keywords", opts.Keywords)
	}
	if opts.Types != "" {
		q.Set("types", opts.Types)
	}
	if opts.Categories != "" {
		q.Set("categories", opts.Categories)
	}
	if opts.WFStatus != "" {
		q.Set("wfstatus", opts.WFStatus)
	}
	if opts.Published != "" {
		q.Set("published", opts.Published)
	}
	if opts.Deleted {
		q.Set("deleted", "true")
	}
	if opts.Owner != "" {
		q.Set("owner", opts.Owner)
	}
	if opts.UserOrg != "" {
		q.Set("userorganisation", opts.UserOrg)
	}
	if opts.TRCID != "" {
		q.Set("trcid", opts.TRCID)
	}
	if opts.ExternalID != "" {
		q.Set("externalid", opts.ExternalID)
	}
	if opts.Language != "" {
		q.Set("lang", opts.Language)
	}
	if opts.UpdatedSince != "" {
		q.Set("lastupdated", opts.UpdatedSince)
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.Asc {
		q.Set("sortorder", "asc")
	}
	if opts.Size > 0 {
		q.Set("size", fmt.Sprintf("%d", opts.Size))
	}
	if opts.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	}

	return q
}

// ListEvents returns events matching the given options.
func (c *Client) ListEvents(opts EventListOptions) (*SearchResult, error) {
	q := buildListQuery(opts.ListOptions)

	if opts.DateFrom != "" {
		q.Set("eventDateRangeStart", opts.DateFrom)
	}
	if opts.DateTo != "" {
		q.Set("eventDateRangeEnd", opts.DateTo)
	}
	if opts.LocationID != "" {
		q.Set("locationId", opts.LocationID)
	}
	if opts.City != "" {
		q.Set("city", opts.City)
	}
	if opts.GeoLat != "" && opts.GeoLon != "" {
		q.Set("geo", opts.GeoLat+","+opts.GeoLon)
	}
	if opts.GeoDistance != "" {
		q.Set("geodistance", opts.GeoDistance)
	}

	endpoint := "/events?" + q.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

// ListLocations returns locations matching the given options.
func (c *Client) ListLocations(opts ListOptions) (*SearchResult, error) {
	q := buildListQuery(opts)
	endpoint := "/locations?" + q.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

// ListRoutes returns routes matching the given options.
func (c *Client) ListRoutes(opts ListOptions) (*SearchResult, error) {
	q := buildListQuery(opts)
	endpoint := "/routes?" + q.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

// ListVenues returns venues matching the given options.
func (c *Client) ListVenues(opts ListOptions) (*SearchResult, error) {
	q := buildListQuery(opts)
	endpoint := "/venues?" + q.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

// ListEventGroups returns event groups matching the given options.
func (c *Client) ListEventGroups(opts ListOptions) (*SearchResult, error) {
	q := buildListQuery(opts)
	endpoint := "/eventgroups?" + q.Encode()
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

// ExportOptions holds parameters specific to Excel export.
type ExportOptions struct {
	// PropertyIDs is a comma-separated list of category property IDs to include as
	// additional columns in the Excel export. Each property ID maps to a category in
	// trcItemCategories; its value is added as a column. Use 'dictionary categories'
	// to find available IDs.
	PropertyIDs string
	// Format is the export format: "excel" (default) or "uitkrant" (events only, plain text).
	Format string
}

// ExportEvents exports events as an Excel file. Supports all list filters plus
// export_propertyids for custom category property columns.
func (c *Client) ExportEvents(opts EventListOptions, exportOpts ExportOptions) ([]byte, error) {
	q := buildListQuery(opts.ListOptions)
	format := exportOpts.Format
	if format == "" {
		format = "excel"
	}
	q.Set("format", format)

	if exportOpts.PropertyIDs != "" {
		q.Set("export_propertyids", exportOpts.PropertyIDs)
	}

	if opts.DateFrom != "" {
		q.Set("eventDateRangeStart", opts.DateFrom)
	}
	if opts.DateTo != "" {
		q.Set("eventDateRangeEnd", opts.DateTo)
	}
	if opts.LocationID != "" {
		q.Set("locationId", opts.LocationID)
	}
	if opts.City != "" {
		q.Set("city", opts.City)
	}
	if opts.GeoLat != "" && opts.GeoLon != "" {
		q.Set("geo", opts.GeoLat+","+opts.GeoLon)
	}
	if opts.GeoDistance != "" {
		q.Set("geodistance", opts.GeoDistance)
	}

	endpoint := "/events?" + q.Encode()
	return c.doRequest("GET", endpoint, nil)
}

// ExportLocations exports locations as an Excel file. Supports all list filters plus
// export_propertyids for custom category property columns.
func (c *Client) ExportLocations(opts ListOptions, exportOpts ExportOptions) ([]byte, error) {
	q := buildListQuery(opts)
	q.Set("format", "excel")

	if exportOpts.PropertyIDs != "" {
		q.Set("export_propertyids", exportOpts.PropertyIDs)
	}

	endpoint := "/locations?" + q.Encode()
	return c.doRequest("GET", endpoint, nil)
}

// ExportVenues exports venues as an Excel file. Supports all list filters plus
// export_propertyids for custom category property columns.
// Note: the API uses "export_properyids" (typo in the API) for venues.
func (c *Client) ExportVenues(opts ListOptions, exportOpts ExportOptions) ([]byte, error) {
	q := buildListQuery(opts)
	q.Set("format", "excel")

	if exportOpts.PropertyIDs != "" {
		// Venues API has a typo: "propery" instead of "property"
		q.Set("export_properyids", exportOpts.PropertyIDs)
	}

	endpoint := "/venues?" + q.Encode()
	return c.doRequest("GET", endpoint, nil)
}

// ExportRoutes exports routes as an Excel file. Supports all list filters.
func (c *Client) ExportRoutes(opts ListOptions) ([]byte, error) {
	q := buildListQuery(opts)
	q.Set("format", "excel")

	endpoint := "/routes?" + q.Encode()
	return c.doRequest("GET", endpoint, nil)
}

// ExportEventGroups exports event groups as an Excel file. Supports all list filters.
func (c *Client) ExportEventGroups(opts ListOptions) ([]byte, error) {
	q := buildListQuery(opts)
	q.Set("format", "excel")

	endpoint := "/eventgroups?" + q.Encode()
	return c.doRequest("GET", endpoint, nil)
}

// GetResource returns a single resource by type and ID.
func (c *Client) GetResource(resourceType, id string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/%s/%s", resourceType, url.PathEscape(id))
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// UpdateResource updates a resource via PUT with the given body.
func (c *Client) UpdateResource(resourceType, id string, data json.RawMessage) error {
	endpoint := fmt.Sprintf("/%s/%s", resourceType, url.PathEscape(id))
	_, err := c.doRequest("PUT", endpoint, bytes.NewReader(data))
	return err
}

// DeleteResource deletes a resource by type and ID.
func (c *Client) DeleteResource(resourceType, id string) error {
	endpoint := fmt.Sprintf("/%s/%s", resourceType, url.PathEscape(id))
	_, err := c.doRequest("DELETE", endpoint, nil)
	return err
}

// PublishResource sets published=true on a resource.
func (c *Client) PublishResource(resourceType, id string) error {
	return c.setPublished(resourceType, id, true)
}

// UnpublishResource sets published=false on a resource.
func (c *Client) UnpublishResource(resourceType, id string) error {
	return c.setPublished(resourceType, id, false)
}

func (c *Client) setPublished(resourceType, id string, published bool) error {
	// GET current resource
	body, err := c.GetResource(resourceType, id)
	if err != nil {
		return fmt.Errorf("getting resource: %w", err)
	}

	// Parse into generic map
	var resource map[string]interface{}
	if err := json.Unmarshal(body, &resource); err != nil {
		return fmt.Errorf("parsing resource: %w", err)
	}

	// Set published field
	resource["published"] = published

	// PUT back
	data, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("marshaling resource: %w", err)
	}

	return c.UpdateResource(resourceType, id, data)
}

// GetComments returns comments for a resource.
func (c *Client) GetComments(resourceType, id string) ([]byte, error) {
	endpoint := fmt.Sprintf("/%s/%s/comments", resourceType, url.PathEscape(id))
	return c.doRequest("GET", endpoint, nil)
}

// AddComment adds a comment to a resource.
func (c *Client) AddComment(resourceType, id, message string) error {
	payload := map[string]string{"text": message}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s/comments", resourceType, url.PathEscape(id))
	_, err = c.doRequest("POST", endpoint, bytes.NewReader(data))
	return err
}

// GetRevisions returns revision history for a resource.
func (c *Client) GetRevisions(resourceType, id string) ([]byte, error) {
	endpoint := fmt.Sprintf("/%s/%s/revisions", resourceType, url.PathEscape(id))
	return c.doRequest("GET", endpoint, nil)
}

// GetAccountMe returns info about the current user.
func (c *Client) GetAccountMe() ([]byte, error) {
	return c.doRequest("GET", "/accounts/me", nil)
}

// ListAccounts returns available accounts.
func (c *Client) ListAccounts() ([]byte, error) {
	return c.doRequest("GET", "/accounts", nil)
}

// GetAccountData returns the first account object as a generic map.
// Keywords, markers, ontology and categories are stored on the account.
func (c *Client) GetAccountData() (map[string]interface{}, error) {
	body, err := c.ListAccounts()
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing accounts: %w", err)
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no accounts found")
	}

	var account map[string]interface{}
	if err := json.Unmarshal(result.Results[0], &account); err != nil {
		return nil, fmt.Errorf("parsing account: %w", err)
	}
	return account, nil
}

// GetKeywords returns keywords for a resource type from the account data.
// The account stores keywords as {type}Keywords (e.g. eventKeywords, locationKeywords).
func (c *Client) GetKeywords(resourceType string) (json.RawMessage, error) {
	account, err := c.GetAccountData()
	if err != nil {
		return nil, err
	}

	key := resourceType + "Keywords"
	val, ok := account[key]
	if !ok {
		return nil, fmt.Errorf("no keywords field %q found on account", key)
	}

	data, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("marshaling keywords: %w", err)
	}
	return data, nil
}

// GetMarkers returns markers for a resource type from the account data.
// The account stores markers as {type}Markers (e.g. eventMarkers, locationMarkers).
func (c *Client) GetMarkers(resourceType string) (json.RawMessage, error) {
	account, err := c.GetAccountData()
	if err != nil {
		return nil, err
	}

	key := resourceType + "Markers"
	val, ok := account[key]
	if !ok {
		return nil, fmt.Errorf("no markers field %q found on account", key)
	}

	data, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("marshaling markers: %w", err)
	}
	return data, nil
}

// GetOntology returns the categorization ontology from the account data.
func (c *Client) GetOntology() (json.RawMessage, error) {
	account, err := c.GetAccountData()
	if err != nil {
		return nil, err
	}

	val, ok := account["categorizationOntology"]
	if !ok {
		return nil, fmt.Errorf("no categorizationOntology found on account")
	}

	data, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("marshaling ontology: %w", err)
	}
	return data, nil
}

// ParseResources parses raw JSON results into Resource structs.
func ParseResources(raw []json.RawMessage) ([]Resource, error) {
	resources := make([]Resource, 0, len(raw))
	for _, r := range raw {
		var res Resource
		if err := json.Unmarshal(r, &res); err != nil {
			return nil, fmt.Errorf("parsing resource: %w", err)
		}
		resources = append(resources, res)
	}
	return resources, nil
}

// resourceTypeToEndpoint returns the API endpoint for a given resource display type.
func ResourceTypeToEndpoint(resourceType string) string {
	switch strings.ToLower(resourceType) {
	case "event", "events":
		return "events"
	case "location", "locations":
		return "locations"
	case "route", "routes":
		return "routes"
	case "venue", "venues":
		return "venues"
	case "eventgroup", "eventgroups":
		return "eventgroups"
	default:
		return resourceType
	}
}
