package main

import (
	"fmt"
	gocli "github.com/mikogs/lib-go-cli"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

const VERSION = "0.1.0"

// DEFAULT_SLEEP is used when reading config fails and the program is set to ignore the error
const DEFAULT_SLEEP = 10

type Config struct {
	Version string `yaml:"version"`
	DB      string `yaml:"database"`
	Orgs    []Org  `yaml:"orgs"`
	DryRun  bool   `yaml:"dry_run"`
	RunOnce bool   `yaml:"run_once"`
	Sleep   int    `yaml:"sleep"`
	// TODO: add reading config map instead
}

func (c *Config) SetFromYAMLFile(f string) error {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return fmt.Errorf("Cannot read config file %s: %w", f, err)
	}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return fmt.Errorf("Cannot unmarshal config yaml: %w", err)
	}
	if c.DB == "" {
		return fmt.Errorf("'database' in config yaml is missing", err)
	}
	fi, err := os.Stat(c.DB)
	if os.IsNotExist(err) {
		return fmt.Errorf("Database file from config.yaml does not exist", err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("Database file from config.yaml is not a regular file", err)
	}
	return nil
}

type Org struct {
	ID      int    `yaml:"id"`
	Viewers []User `yaml:"viewers"`
	Editors []User `yaml:"editors"`
	Admins  []User `yaml:"admins"`
}

type User struct {
	Login string `yaml:"login"`
}

func main() {
	cli := gocli.NewCLI("grafana-sidecar-users-yaml", "Updates Grafana user org role from file", "Mikolaj Gasior <miko@dsp.gs>")
	cmdStart := cli.AddCmd("start", "Starts the daemon", startHandler)
	cmdStart.AddFlag("config", "c", "config", "YAML file with users", gocli.TypePathFile|gocli.MustExist|gocli.Required, nil)
	cmdStart.AddFlag("quiet", "q", "", "Quite mode. Do not output anything", gocli.TypeBool, nil)
	cmdStart.AddFlag("ignore_errors", "i", "", "Ignore errors and continue", gocli.TypeBool, nil)
	_ = cli.AddCmd("version", "Prints version", versionHandler)
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"App", "version"}
	}
	os.Exit(cli.Run(os.Stdout, os.Stderr))
}

func versionHandler(c *gocli.CLI) int {
	fmt.Fprintf(os.Stdout, VERSION+"\n")
	return 0
}

func startHandler(c *gocli.CLI) int {
	cfg := Config{}

	ch := make(chan int)
	go func(ch chan int) {
		for {
			fmt.Fprintf(os.Stdout, "Reading config file %s...\n", c.Flag("config"))
			err := cfg.SetFromYAMLFile(c.Flag("config"))
			if err != nil {

				fmt.Fprintf(os.Stderr, "Error with config file: %v\n", err.Error())
				if c.Flag("ignore_errors") == "false" || cfg.RunOnce {
					ch <- 1
				} else {
					fmt.Fprintf(os.Stderr, "Ignoring error and continuing to do nothing...\n")
					cfg.Sleep = DEFAULT_SLEEP
					goto SLEEP
				}
			}

			if cfg.DryRun {
				fmt.Fprintf(os.Stdout, "Dry-run is set. No changes will be made\n")
			}
			for _, org := range cfg.Orgs {
				fmt.Fprintf(os.Stdout, "Got org %v from the config file\n", org.ID)
				for _, viewer := range org.Viewers {
					fmt.Fprintf(os.Stdout, "Setting login '%s' to Viewer for org %d...\n", viewer.Login, org.ID)
				}
				for _, editor := range org.Editors {
					fmt.Fprintf(os.Stdout, "Setting login '%s' to Editor for org %d...\n", editor.Login, org.ID)
				}
				for _, admin := range org.Admins {
					fmt.Fprintf(os.Stdout, "Setting login '%s' to Admin for org %d...\n", admin.Login, org.ID)
				}
			}
		SLEEP:
			if cfg.RunOnce {
				break
			} else {
				fmt.Fprintf(os.Stdout, "Sleeping %d seconds...\n", cfg.Sleep)
				time.Sleep(time.Duration(cfg.Sleep) * time.Second)
			}
		}
		ch <- 0
	}(ch)
	lastErr := <-ch
	return lastErr
}
