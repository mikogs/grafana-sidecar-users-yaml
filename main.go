package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	gocli "github.com/mikogs/lib-go-cli"
	"os"
	"time"
)

// DEFAULT_SLEEP is used when reading config fails and the program is set to ignore the error
const DEFAULT_SLEEP = 10

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

func readConfig(f string, cfg *Config, ig string, once bool) int {
	fmt.Fprintf(os.Stdout, "Reading config file %s...\n", f)
	err := cfg.SetFromYAMLFile(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with config file: %v\n", err.Error())
		if ig == "false" || once {
			return -1
		} else {
			fmt.Fprintf(os.Stderr, "Ignoring error and continuing to do nothing...\n")
			return 1
		}
	}
	return 0
}

func connectToDB(dbfile string, dry bool, ig string, once bool) (int, *sql.DB) {
	var err error
	var db *sql.DB
	if dry {
		fmt.Fprintf(os.Stdout, "Dry-run is set. No changes will be made\n")
	} else {
		db, err = sql.Open("sqlite3", dbfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error with connecting to the database: %v\n", err.Error())
			if ig == "false" || once {
				return -1, nil
			} else {
				fmt.Fprintf(os.Stderr, "Ignoring error and continuing to do nothing...\n")
				return 1, nil
			}
		}
	}
	return 0, db

}

func update(role string, login string, org int, db *sql.DB, ig string, once bool) int {
	fmt.Fprintf(os.Stdout, "Setting login '%s' to %s for org %d...\n", login, role, org)
	if _, err := db.Exec("UPDATE org_user SET role = ? WHERE user_id IN (SELECT id FROM user WHERE login=?);", role, login); err != nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "UPDATE query for Viewer failed to execute: %v\n", err.Error())
			if ig == "false" || once {
				return -1
			} else {
				fmt.Fprintf(os.Stderr, "Ignoring error and continuing to do nothing...\n")
				return 1
			}
		}
	}
	return 0

}

func startHandler(c *gocli.CLI) int {
	cfg := Config{}
	var db *sql.DB
	ch := make(chan int)
	go func(ch chan int) {
		for {
			r := readConfig(c.Flag("config"), &cfg, c.Flag("ignore_errors"), cfg.RunOnce)
			if r == -1 {
				ch <- 1
			}
			if r == 1 {
				cfg.Sleep = DEFAULT_SLEEP
				goto SLEEP
			}

			r, db = connectToDB(cfg.DB, cfg.DryRun, c.Flag("ignore_errors"), cfg.RunOnce)
			if r == -1 {
				ch <- 1
			}
			if r == 1 {
				cfg.Sleep = DEFAULT_SLEEP
				goto SLEEP
			}

			for _, org := range cfg.Orgs {
				fmt.Fprintf(os.Stdout, "Got org %v from the config file\n", org.ID)
				for _, viewer := range org.Viewers {
					r := update("Viewer", viewer.Login, org.ID, db, c.Flag("ignore_errors"), cfg.RunOnce)
					if r == -1 {
						ch <- 1
					}
					if r == 1 {
						cfg.Sleep = DEFAULT_SLEEP
						goto SLEEP
					}
				}
				for _, editor := range org.Editors {
					r := update("Editor", editor.Login, org.ID, db, c.Flag("ignore_errors"), cfg.RunOnce)
					if r == -1 {
						ch <- 1
					}
					if r == 1 {
						cfg.Sleep = DEFAULT_SLEEP
						goto SLEEP
					}

				}
				for _, admin := range org.Admins {
					r := update("Admin", admin.Login, org.ID, db, c.Flag("ignore_errors"), cfg.RunOnce)
					if r == -1 {
						ch <- 1
					}
					if r == 1 {
						cfg.Sleep = DEFAULT_SLEEP
						goto SLEEP
					}

				}
			}

			if !cfg.DryRun {
				db.Close()
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
