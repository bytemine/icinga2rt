package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/filter"
)

type localFilterConfig struct {
	Any filter.Any
	All filter.All
}

type icingaConfig struct {
	URL         string
	User        string
	Password    string
	Filter      string
	LocalFilter localFilterConfig
	Insecure    bool
	Retries     int
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
	Mappings     string
	mappings     []mapping
	Nobody       string
	Queue        string
	ClosedStatus []string
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
		Filter:   "",
		LocalFilter: localFilterConfig{
			All: filter.All{filter.Filter{Users: []string{"request-tracker"}}},
		},
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
		Mappings: "/etc/bytemine/icinga2rt.csv",
		mappings: []mapping{},
		Nobody:   "Nobody",
		Queue:    "general",
		ClosedStatus: []string{
			"done",
			"resolved",
			"deleted",
		},
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

	if conf.Icinga.LocalFilter.All != nil && conf.Icinga.LocalFilter.Any != nil {
		return fmt.Errorf("Only Icinga.LocalFilter.All or Icinga.LocalFilter.Any can be set")
	}

	if conf.Ticket.Queue == "" {
		return fmt.Errorf("Ticket.Queue must be set.")
	}

	if conf.Ticket.Nobody == "" {
		return fmt.Errorf("Ticket.Nobody must be set.")
	}

	if conf.Ticket.Mappings == "" || len(conf.Ticket.Mappings) == 0 {
		return fmt.Errorf("Ticket.Mappings must be set.")
	}

	if conf.Ticket.ClosedStatus == nil || len(conf.Ticket.ClosedStatus) == 0 {
		return fmt.Errorf("Ticket.ClosedStatus must be set.")
	}

	if conf.Cache.File == "" {
		return fmt.Errorf("Cache.File must be set.")
	}

	return nil
}

func loadConfig(filename string) (*config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return readConfig(f)
}

func readConfig(r io.Reader) (*config, error) {
	var c config

	dec := json.NewDecoder(r)

	err := dec.Decode(&c)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(c.Ticket.Mappings)
	if err != nil {
		return nil, err
	}

	mappings, err := readMappings(f)
	if err != nil {
		return nil, err
	}

	c.Ticket.mappings = mappings

	return &c, nil
}

func saveConfig(filename string, c *config) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	return writeConfig(f, c)
}

func writeConfig(w io.Writer, c *config) error {
	x, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}

	_, err = w.Write(x)
	if err != nil {
		return err
	}

	return nil
}

func parseCSVBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %v", value)
	}
}

const (
	actionStringDelete  = "delete"
	actionStringComment = "comment"
	actionStringCreate  = "create"
	actionStringIgnore  = "ignore"
	actionStringStatus  = "status"
)

func parseCSVAction(action []string) (actionFunc, error) {
	switch len(action) {
	case 1:
		switch strings.ToLower(action[0]) {
		case actionStringDelete:
			return (*ticketUpdater).delete, nil
		case actionStringComment:
			return (*ticketUpdater).comment, nil
		case actionStringCreate:
			return (*ticketUpdater).create, nil
		case actionStringIgnore:
			return (*ticketUpdater).ignore, nil
		default:
			return nil, fmt.Errorf("invalid action value: %v", action[0])
		}
	case 3:
		switch strings.ToLower(action[0]) {
		case actionStringStatus:
			status := action[1]
			invalidate, err := parseCSVBool(action[2])
			if err != nil {
				return nil, fmt.Errorf("invalid action: %v: %v", action, err)
			}
			return statusActionFunc(status, invalidate), nil
		default:
			return nil, fmt.Errorf("invalid action: %v, three valued action has to be %v", action[0], actionStringStatus)
		}
	default:
			return nil, fmt.Errorf("invalid action: %v, either one or three fields required", action)
	}
}

func readMappings(r io.Reader) ([]mapping, error) {
	ms := []mapping{}

	x := csv.NewReader(r)
	x.Comment = '#'

	x.FieldsPerRecord = -1
	line := 0
	for {
		line++
		record, err := x.Read()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}

			break
		}

		if len(record) < 4 {
			return nil, fmt.Errorf("error in line %v: to few fields, have %v want 4", line, len(record))
		}

		// uppercase the value as icingas strings are uppercase
		state := event.NewState(strings.ToUpper(record[0]))
		if state == event.StateNil {
			return nil, fmt.Errorf("error in line %v: invalid state value %v", line, record[0])
		}

		oldState := event.NewState(record[1])

		owned, err := parseCSVBool(record[2])
		if err != nil {
			return nil, fmt.Errorf("error in line %v: %v", line, err)
		}

		action, err := parseCSVAction(record[3:])
		if err != nil {
			return nil, fmt.Errorf("error in line %v: %v", line, err)
		}

		m := mapping{condition: condition{state: state, oldState: oldState, owned: owned}, action: action}
		ms = append(ms, m)
	}

	return ms, nil
}
