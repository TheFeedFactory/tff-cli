package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	Token string
}

func ConfigLocations() []string {
	locations := []string{
		".env",
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		locations = append(locations, filepath.Join(homeDir, ".config", "tff-cli", ".env"))
	}

	return locations
}

func Load(configFile string) (*Config, error) {
	if configFile != "" {
		if err := godotenv.Load(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
	} else {
		for _, loc := range ConfigLocations() {
			if _, err := os.Stat(loc); err == nil {
				_ = godotenv.Load(loc)
				break
			}
		}
	}

	token := os.Getenv("FF_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("FF_ACCESS_TOKEN not set.\n\n%s", configHelp())
	}

	return &Config{Token: token}, nil
}

func configHelp() string {
	return `To configure the FeedFactory CLI, set your access token using one of these methods:

1. Environment variable:
   export FF_ACCESS_TOKEN=your-token-here

2. .env file in the current directory:
   FF_ACCESS_TOKEN=your-token-here

3. Config file at ~/.config/tff-cli/.env:
   FF_ACCESS_TOKEN=your-token-here

4. Command line flag:
   tff --token your-token-here <command>

Run 'tff configure' for more information.`
}

func PrintConfigHelp() {
	fmt.Println(`FeedFactory CLI Configuration
=============================

The CLI needs an access token to authenticate with the FeedFactory API.

Getting your access token:
  1. Log in to https://app.thefeedfactory.nl
  2. Go to your account settings
  3. Generate or copy your API access token

Configuration methods (in order of precedence):
  1. --token flag:      tff --token <token> events list
  2. Environment var:   export FF_ACCESS_TOKEN=<token>
  3. .env file:         Create a .env file with FF_ACCESS_TOKEN=<token>

Config file locations (first found wins):
  - .env (current directory)
  - ~/.config/tff-cli/.env

Example .env file:
  FF_ACCESS_TOKEN=your-access-token-here`)
}
