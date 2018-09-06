package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytemine/go-icinga2/event"
)

type icingaConfig struct {
	URL      string
	User     string
	Password string
	Insecure bool
	Retries  int
}

type rtConfig struct {
	URL      string
	User     string
	Password string
	Insecure bool
}

type cacheConfig struct {
	File string
}

type ticketConfig struct {
	Mappings []Mapping
	Nobody   string
	Queue    string
}

type config struct {
	Icinga icingaConfig
	RT     rtConfig
	Cache  cacheConfig
	Ticket ticketConfig
}

var defaultConfig = config{
	Icinga: icingaConfig{
		URL:      "https://monitoring.example.com:5665",
		User:     "root",
		Password: "secret",
		Insecure: true,
		Retries:  5,
	},
	RT: rtConfig{
		URL:      "https://support.example.com",
		User:     "apiuser",
		Password: "secret",
		Insecure: true,
	},
	Cache: cacheConfig{
		File: "/var/lib/icinga2rt/icinga2rt.bolt",
	},
	Ticket: ticketConfig{
		Mappings: []Mapping{
			Mapping{
				Condition: Condition{
					State:    "OK",
					state:    event.StateOK,
					Existing: false,
					Owned:    false,
				},
				Action: "ignore",
				action: (*ticketUpdater).ignore,
			},
			Mapping{
				Condition: Condition{
					State:    "OK",
					state:    event.StateOK,
					Existing: true,
					Owned:    false,
				},
				Action: "delete",
				action: (*ticketUpdater).delete,
			},
			Mapping{
				Condition: Condition{
					State:    "OK",
					state:    event.StateOK,
					Existing: true,
					Owned:    true,
				},
				Action: "comment",
				action: (*ticketUpdater).comment,
			},
			Mapping{
				Condition: Condition{
					State:    "WARNING",
					state:    event.StateWarning,
					Existing: false,
					Owned:    false,
				},
				Action: "create",
				action: (*ticketUpdater).create,
			},
			Mapping{
				Condition: Condition{
					State:    "WARNING",
					state:    event.StateWarning,
					Existing: true,
					Owned:    false,
				},
				Action: "comment",
				action: (*ticketUpdater).comment,
			},
			Mapping{
				Condition: Condition{
					State:    "CRITICAL",
					state:    event.StateCritical,
					Existing: false,
					Owned:    false,
				},
				Action: "create",
				action: (*ticketUpdater).create,
			},
			Mapping{
				Condition: Condition{
					State:    "CRITICAL",
					state:    event.StateCritical,
					Existing: true,
					Owned:    false,
				},
				Action: "comment",
				action: (*ticketUpdater).create,
			},
			Mapping{
				Condition: Condition{
					State:    "UNKNOWN",
					state:    event.StateUnknown,
					Existing: false,
					Owned:    false,
				},
				Action: "create",
				action: (*ticketUpdater).create,
			},
			Mapping{
				Condition: Condition{
					State:    "UNKNOWN",
					state:    event.StateUnknown,
					Existing: true,
					Owned:    false,
				},
				Action: "comment",
				action: (*ticketUpdater).create,
			},
		},
		Nobody: "Nobody",
		Queue:  "general",
	},
}

func checkConfig(conf *config) error {
	if conf.Icinga.URL == "" {
		return fmt.Errorf("Icinga.URL must be set.")
	}

	if conf.Icinga.User == "" {
		return fmt.Errorf("Icinga.User must be set.")
	}

	if conf.Icinga.Retries == 0 {
		return fmt.Errorf("Icinga.Retries must be > 0.")
	}

	if conf.Ticket.Queue == "" {
		return fmt.Errorf("Ticket.Queue must be set.")
	}

	if conf.Ticket.Nobody == "" {
		return fmt.Errorf("Ticket.Nobody must be set.")
	}

	if conf.Ticket.Mappings == nil || len(conf.Ticket.Mappings) == 0 {
		return fmt.Errorf("Ticket.Mappings must be set.")
	}

	if conf.Cache.File == "" {
		return fmt.Errorf("Cache.File must be set.")
	}

	return nil
}

func readConfig(filename string) (*config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	var c config

	dec := json.NewDecoder(f)

	err = dec.Decode(&c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func writeConfig(filename string, c *config) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	x, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}

	_, err = f.Write(x)
	if err != nil {
		return err
	}

	return nil
}
