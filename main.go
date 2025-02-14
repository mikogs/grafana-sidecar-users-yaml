package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	gocli "github.com/go-phings/broccli"
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
	cli := gocli.NewCLI("grafana-sidecar-users-yaml", "Updates Grafana user org role from file", "Mikolaj Gasior")
	cmdStart := cli.AddCmd("start", "Starts the daemon", startHandler)
	cmdStart.AddFlag("config", "c", "config", "YAML file with users", gocli.TypePathFile, gocli.IsExistent|gocli.IsRequired)
	cmdStart.AddFlag("quiet", "q", "", "Quite mode. Do not output anything", gocli.TypeBool, 0)
	cmdStart.AddFlag("ignore_errors", "i", "", "Ignore errors and continue", gocli.TypeBool, 0)
	_ = cli.AddCmd("version", "Prints version", versionHandler)
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"App", "version"}
	}
	os.Exit(cli.Run())
}

func versionHandler(c *gocli.CLI) int {
	fmt.Fprintf(os.Stdout, VERSION+"\n")
	return 0
}

func readConfig(f string, cfg *Config) error {
	fmt.Fprintf(os.Stdout, "Reading config file %s...\n", f)
	err := cfg.SetFromYAMLFile(f)
	if err != nil {
		return fmt.Errorf("Error with config file: %w\n", err)
	}
	return nil
}

func connectToDB(cfg *Config) (*sql.DB, error) {
	if cfg.DryRun {
		fmt.Fprintf(os.Stderr, "Dry-running...\n")
		return nil, nil
	}
	db, err := sql.Open("sqlite3", cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("Error with connecting to the database: %w\n", err)
	}
	return db, err

}

func update(role string, login string, org int, db *sql.DB, cfg *Config) error {
	fmt.Fprintf(os.Stdout, "Setting login '%s' to %s for org %d...\n", login, role, org)

	if cfg.DryRun {
		return nil
	}

	if _, err := db.Exec("UPDATE org_user SET role = ? WHERE user_id IN (SELECT id FROM user WHERE login=?);", role, login); err != nil {
		return fmt.Errorf("UPDATE query for Viewer failed to execute: %w", err)
	}
	return nil

}

func updateOrgs(cfg *Config, db *sql.DB) error {
	for _, org := range cfg.Orgs {
		fmt.Fprintf(os.Stdout, "Got org %v from the config file\n", org.ID)
		for _, viewer := range org.Viewers {
			err := update("Viewer", viewer.Login, org.ID, db, cfg)
			if err != nil {
				return err
			}
		}
		for _, editor := range org.Editors {
			err := update("Editor", editor.Login, org.ID, db, cfg)
			if err != nil {
				return err
			}
		}
		for _, admin := range org.Admins {
			err := update("Admin", admin.Login, org.ID, db, cfg)
			if err != nil {
				return err
			}
		}
	}
	return nil

}

func startHandler(c *gocli.CLI) int {
	ch := make(chan int)
	go func(ch chan int) {
		var cfg Config
		var db *sql.DB
		for {
			err := readConfig(c.Flag("config"), &cfg)
			if err != nil {
				cfg.Sleep = DEFAULT_SLEEP
			}
			if err == nil {
				db, err = connectToDB(&cfg)
			}
			if err == nil {
				err = updateOrgs(&cfg, db)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err.Error())
				if c.Flag("ignore_errors") == "false" || cfg.RunOnce {
					ch <- 1
				} else {
					fmt.Fprintf(os.Stderr, "Ignoring error and continuing to do nothing...\n")
				}
			}

			if !cfg.DryRun {
				db.Close()
			}
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
