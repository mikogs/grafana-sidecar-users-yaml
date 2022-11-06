package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

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
