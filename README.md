# tff - FeedFactory CLI

A command-line interface for the [FeedFactory API](https://app.thefeedfactory.nl). Manage events, locations, routes, venues, and event groups directly from your terminal.

Built for both human operators and LLM-driven automation, with verbose help descriptions and structured JSON output.

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap TheFeedFactory/tap
brew install tff
```

### Go Install

```bash
go install github.com/TheFeedFactory/tff-cli@latest
```

### Download Binary

Download pre-built binaries from the [Releases](https://github.com/TheFeedFactory/tff-cli/releases) page.

### Build from Source

```bash
git clone https://github.com/TheFeedFactory/tff-cli.git
cd tff-cli
go build -o tff .
```

## Configuration

The CLI requires a FeedFactory API access token. Configure it using any of these methods (in order of precedence):

### 1. Command-line flag

```bash
tff --token <your-token> events list
```

### 2. Environment variable

```bash
export FF_ACCESS_TOKEN=<your-token>
```

### 3. `.env` file

Create a `.env` file in your working directory or at `~/.config/tff-cli/.env`:

```
FF_ACCESS_TOKEN=your-access-token-here
```

### Getting your access token

1. Log in to [app.thefeedfactory.nl](https://app.thefeedfactory.nl)
2. Go to your account settings
3. Generate or copy your API access token

Run `tff configure` for setup instructions.

## Quick Start

```bash
# List recent events
tff events list

# Search events in Amsterdam
tff events list -s "Amsterdam"

# List approved locations
tff locations list -w approved

# Get details for a specific venue
tff venues get <venue-id>

# Export events to Excel
tff events export -o events.xlsx

# Show help for any command
tff events list --help
```

## Commands

### Resource Types

The CLI supports five resource types, each with the same set of subcommands:

| Resource | Command | Description |
|----------|---------|-------------|
| Events | `tff events` | Cultural events, festivals, performances, exhibitions |
| Locations | `tff locations` | Physical locations and addresses |
| Routes | `tff routes` | Walking, cycling, and other routes |
| Venues | `tff venues` | Theaters, museums, concert halls |
| Event Groups | `tff eventgroups` | Grouped/recurring event series |

### Subcommands

Each resource type supports these subcommands:

| Subcommand | Description |
|------------|-------------|
| `list` | List and search resources with filtering |
| `get <id>` | Get detailed information about a resource |
| `export` | Export resources to Excel (.xlsx) |
| `delete <id>` | Delete a resource |
| `publish <id>` | Make a resource publicly visible |
| `unpublish <id>` | Hide a resource from public view |
| `comments <id>` | List comments on a resource |
| `comment <id> <msg>` | Add a comment to a resource |
| `revisions <id>` | Show revision history |

### Dictionary Commands

```bash
# List keywords for a resource type
tff dictionary keywords event

# List markers for a resource type
tff dictionary markers location

# Show the full category ontology tree
tff dictionary ontology

# List all leaf categories with IDs (useful for --categories filter)
tff dictionary categories
tff dictionary categories --lang en
```

### Account Commands

```bash
# Show current user info
tff accounts me

# List available accounts/organisations
tff accounts list
```

## Filtering & Search

### Full-text Search

```bash
# Simple text search
tff events list -s "jazz festival"

# Search by keyword tag
tff events list -s "tag:music"

# Search by marker
tff events list -s "marker:featured"
```

### Workflow Status

Filter by workflow status using `-w`:

```bash
tff events list -w draft
tff events list -w approved
tff events list -w readyforvalidation
tff events list -w rejected
tff events list -w archived
```

### Markers & Keywords

```bash
# Filter by markers (comma-separated)
tff events list --markers "featured,highlight"

# Exclude markers with '!' prefix
tff events list --markers "!archived,featured"

# Filter by keywords
tff locations list --keywords "museum,art"
```

### Category & Type Filters

```bash
# Filter by categories
tff events list --categories "2.1.3"

# Filter by types
tff locations list --types "restaurant,hotel"

# Find category IDs
tff dictionary categories --lang nl
```

### Date Filters (Events)

```bash
# Events in the next 2 weeks
tff events list --date-from 0d --date-to 2w

# Events in a specific range
tff events list --date-from 2026-03-01 --date-to 2026-03-31

# Events at a specific location
tff events list --location-id <location-id>

# Events in a city
tff events list --city Amsterdam
```

### Geographic Filtering (Events)

```bash
# Events within 10km of Amsterdam center
tff events list --geo 52.37,4.89 --geo-distance 10km
```

### Updated Since

Supports relative time expressions and absolute dates:

```bash
# Updated in the last 3 days
tff events list --updated-since 3d

# Updated in the last 2 weeks
tff locations list --updated-since 2w

# Updated in the last month
tff venues list --updated-since 1mo

# Updated since a specific date
tff routes list --updated-since 2026-01-15
```

### Sorting & Pagination

```bash
# Sort by title ascending
tff events list -o title --asc

# Get 100 results per page
tff events list -l 100

# Get page 3
tff events list -p 3

# Sort options: modified (default), created, title, wfstatus
```

### Combining Filters

All filters can be combined:

```bash
tff events list \
  -s "music" \
  -w approved \
  --published true \
  --date-from 0d \
  --date-to 3mo \
  --city Amsterdam \
  -o title --asc \
  -l 50
```

## Export

Export resources to Excel spreadsheets (.xlsx). The API generates the file server-side with all resource fields.

### Basic Export

```bash
tff events export -o events.xlsx
tff locations export -o locations.xlsx
tff routes export -o routes.xlsx
tff venues export -o venues.xlsx
tff eventgroups export -o eventgroups.xlsx
```

### Filtered Export

All list filters work with export:

```bash
# Export approved events in Amsterdam
tff events export -o amsterdam-events.xlsx -w approved --city Amsterdam

# Export locations updated this month
tff locations export -o recent.xlsx --updated-since 1mo
```

### Custom Property Columns

Add category property values as extra columns (events, locations, venues):

```bash
# Find property IDs
tff dictionary categories

# Export with custom columns
tff events export -o events.xlsx --export-propertyids "12345,67890"
tff locations export -o locations.xlsx --export-propertyids "12345"
tff venues export -o venues.xlsx --export-propertyids "12345"
```

### Uitkrant Format (Events Only)

Export events as plain text for publication. Requires a date range:

```bash
tff events export -o uitkrant.txt --format uitkrant --date-from 2026-03-01 --date-to 2026-03-31
```

## JSON Output

Add `-j` / `--json` to any command for structured JSON output:

```bash
# Full API response as JSON
tff events list -j

# Pipe to jq for processing
tff events list -j | jq '.results[].id'

# Get a single resource as JSON
tff events get <id> -j
```

## Publishing & Unpublishing

```bash
# Publish a resource (makes it publicly visible)
tff events publish <event-id>
tff locations publish <location-id>

# Unpublish a resource (hides from public)
tff events unpublish <event-id>
```

## Comments & Revisions

```bash
# List comments on an event
tff events comments <event-id>

# Add a comment
tff events comment <event-id> "Reviewed and approved"

# View revision history
tff events revisions <event-id>

# JSON output for comments/revisions
tff events comments <event-id> -j
```

## Deleting Resources

```bash
# Delete with confirmation prompt
tff events delete <event-id>

# Skip confirmation
tff events delete <event-id> -f
```

## Examples

### Daily Workflow

```bash
# Check what was updated today
tff events list --updated-since 1d

# Review draft events
tff events list -w draft

# Approve and publish an event
tff events publish <event-id>
tff events comment <event-id> "Reviewed and published"

# Export this week's events for newsletter
tff events export -o newsletter.xlsx --date-from 0d --date-to 1w -w approved --published true
```

### Batch Operations with Shell

```bash
# Get all draft event IDs
tff events list -w draft -j | jq -r '.results[].id'

# Export all resource types
for type in events locations routes venues eventgroups; do
  tff $type export -o "${type}.xlsx"
done
```

### LLM Integration

The CLI is designed for LLM-driven automation:

```bash
# Structured JSON output for parsing
tff events list -j --size 100

# Verbose help descriptions for tool discovery
tff events list --help

# Category reference data
tff dictionary categories -j
```

## Project Structure

```
tff-cli/
├── main.go                    # Entry point, Kong CLI definition
├── go.mod
├── cmd/
│   ├── events.go              # Events commands
│   ├── locations.go           # Locations commands
│   ├── routes.go              # Routes commands
│   ├── venues.go              # Venues commands
│   ├── eventgroups.go         # Event groups commands
│   ├── dictionary.go          # Dictionary commands (keywords, markers, ontology)
│   ├── accounts.go            # Account commands
│   ├── util.go                # Shared output utilities
│   └── timefilter.go          # Relative date parsing
├── internal/
│   ├── api/
│   │   └── client.go          # HTTP client, all API methods
│   └── config/
│       └── config.go          # Config loading (.env, env vars)
├── .goreleaser.yml            # Release automation
└── README.md
```

## Development

### Prerequisites

- Go 1.25+

### Building

```bash
go build -o tff .
```

### Running

```bash
# With token
./tff --token <token> events list

# With .env file
echo "FF_ACCESS_TOKEN=<token>" > .env
./tff events list
```

### Releasing

Releases are automated with [GoReleaser](https://goreleaser.com/). Tag a version to trigger a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GoReleaser builds binaries for Linux, macOS, and Windows (amd64 + arm64) and updates the Homebrew formula.

## License

MIT
