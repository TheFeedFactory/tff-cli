package cmd

import (
	"github.com/TheFeedFactory/tff-cli/internal/api"
)

type AccountsCmd struct {
	Me   AccountsMeCmd   `cmd:"" help:"Show information about the currently authenticated user, including name, email, role, and organisation."`
	List AccountsListCmd `cmd:"" help:"List all accounts/organisations available to the current user."`
}

type AccountsMeCmd struct {
	JSON bool `short:"j" help:"Output as JSON."`
}

func (c *AccountsMeCmd) Run(client *api.Client) error {
	body, err := client.GetAccountMe()
	if err != nil {
		return err
	}

	return printRawJSON(body)
}

type AccountsListCmd struct {
	JSON bool `short:"j" help:"Output as JSON."`
}

func (c *AccountsListCmd) Run(client *api.Client) error {
	body, err := client.ListAccounts()
	if err != nil {
		return err
	}

	return printRawJSON(body)
}
