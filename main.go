package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/TheFeedFactory/tff-cli/cmd"
	"github.com/TheFeedFactory/tff-cli/internal/api"
	"github.com/TheFeedFactory/tff-cli/internal/config"
)

var version = "0.1.0"

var CLI struct {
	Config string `short:"c" help:"Path to config file (.env format)." type:"path"`
	Token  string `help:"Access token (overrides config file and environment variable)." env:"FF_ACCESS_TOKEN"`

	Events      cmd.EventsCmd      `cmd:"" help:"Manage events (list, get, export, delete, publish, unpublish, comments, revisions)."`
	Locations   cmd.LocationsCmd   `cmd:"" help:"Manage locations (list, get, export, delete, publish, unpublish, comments, revisions)."`
	Routes      cmd.RoutesCmd      `cmd:"" help:"Manage routes (list, get, export, delete, publish, unpublish, comments, revisions)."`
	Venues      cmd.VenuesCmd      `cmd:"" help:"Manage venues (list, get, export, delete, publish, unpublish, comments, revisions)."`
	EventGroups cmd.EventGroupsCmd `cmd:"" name:"eventgroups" help:"Manage event groups (list, get, export, delete, publish, unpublish, comments, revisions)."`
	Dictionary  cmd.DictionaryCmd  `cmd:"" help:"Dictionary reference data (keywords, markers, ontology, categories)."`
	Accounts    cmd.AccountsCmd    `cmd:"" help:"Account information (me, list)."`
	Configure   ConfigureCmd       `cmd:"" help:"Show configuration help and setup instructions. Does not require authentication."`
}

type ConfigureCmd struct{}

func (c *ConfigureCmd) Run() error {
	config.PrintConfigHelp()
	return nil
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf("tff v%s\n", version)
			return
		}
	}

	ctx := kong.Parse(&CLI,
		kong.Name("tff"),
		kong.Description("FeedFactory CLI - A command-line interface for the FeedFactory API (v"+version+")"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	switch ctx.Command() {
	case "configure":
		err := ctx.Run()
		ctx.FatalIfErrorf(err)
		return
	}

	cfg, err := config.Load(CLI.Config)
	if err != nil {
		if CLI.Token == "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cfg = &config.Config{Token: CLI.Token}
	}
	if CLI.Token != "" {
		cfg.Token = CLI.Token
	}

	client := api.NewClient(cfg)

	err = ctx.Run(client)
	ctx.FatalIfErrorf(err)
}
