package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/rt"
)

const (
	actionStringDelete  = "delete"
	actionStringComment = "comment"
	actionStringCreate  = "create"
	actionStringIgnore  = "ignore"
)

type actionFunc func(*ticketUpdater, *event.Notification) error

// Condition describes the properties an event must have to match.
type Condition struct {
	state    event.State
	existing bool
	owned    bool
}

type jsonCondition struct {
	State    string
	Existing bool
	Owned    bool
}

// UnmarshalJSON unmarshals JSON into a condition.
func (c *Condition) UnmarshalJSON(data []byte) error {
	var x jsonCondition
	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}

	switch x.State {
	case "OK":
		c.state = event.StateOK
	case "WARNING":
		c.state = event.StateWarning
	case "CRITICAL":
		c.state = event.StateCritical
	case "UNKNOWN":
		c.state = event.StateUnknown
	default:
		return fmt.Errorf("unknown state: \"%v\"", x.State)
	}

	c.existing = x.Existing
	c.owned = x.Owned

	return nil
}

// MarshalJSON marshals a Condition to JSON.
func (c Condition) MarshalJSON() ([]byte, error) {
	x := jsonCondition{State: c.state.String(), Existing: c.existing, Owned: c.owned}

	b, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Mapping describes how an event matching Condition should be acted upon.
type Mapping struct {
	condition  Condition
	action     actionFunc
	actionName string // helper for marshalling as we can't compare functions
}

type jsonMapping struct {
	Condition Condition
	Action    string
}

// UnmarshalJSON unmarshals a Mapping from JSON.
func (m *Mapping) UnmarshalJSON(data []byte) error {
	var x jsonMapping
	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}

	switch x.Action {
	case actionStringDelete:
		m.action = (*ticketUpdater).delete
	case actionStringComment:
		m.action = (*ticketUpdater).comment
	case actionStringCreate:
		m.action = (*ticketUpdater).create
	case actionStringIgnore:
		m.action = (*ticketUpdater).ignore
	default:
		return fmt.Errorf("unknown action: %v", m.action)
	}

	m.condition = x.Condition

	return nil
}

// MarshalJSON marshals a Mapping to JSON.
func (m Mapping) MarshalJSON() ([]byte, error) {
	x := jsonMapping{Condition: m.condition, Action: m.actionName}

	b, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type ticketUpdater struct {
	cache    *cache
	rtClient *rt.Client
	mappings []Mapping
	nobody   string
	queue    string
}

func newTicketUpdater(cache *cache, rtClient *rt.Client, mappings []Mapping, nobody string, queue string) *ticketUpdater {
	return &ticketUpdater{cache: cache, rtClient: rtClient, mappings: mappings, nobody: nobody, queue: queue}
}

func (t *ticketUpdater) update(e *event.Notification) error {
	if *debug {
		log.Printf("ticket updater: new event %v", formatEventSubject(e))
	}

	oldEvent, ticketID, err := t.cache.getEventTicket(e)
	if err != nil {
		return err
	}

	existing := false
	owned := false

	if oldEvent != nil {
		existing = true

		oldTicket, err := t.rtClient.Ticket(ticketID)
		if err != nil {
			return err
		}

		owned = oldTicket.Owner == t.nobody
	}

	for _, v := range t.mappings {
		x := Condition{
			state:    e.CheckResult.State,
			existing: existing,
			owned:    owned,
		}

		if *debug {
			log.Printf("ticket updater: matching\nCondition: %#v\nEvent:     %#v", v.condition, x)
		}

		if v.condition == x {
			if *debug {
				log.Printf("ticket updater: matched %v", v.action)
			}

			return v.action(t, e)
		}
	}

	return nil
}

func (t *ticketUpdater) delete(e *event.Notification) error {
	_, ticketID, err := t.cache.getEventTicket(e)
	if err != nil {
		return err
	}

	newTicket := &rt.Ticket{ID: ticketID, Status: "deleted"}

	updatedTicket, err := t.rtClient.UpdateTicket(newTicket)
	if err != nil {
		return err
	}

	if *debug {
		log.Printf("ticket updater: deleted ticket #%v", updatedTicket.ID)
	}

	if err = t.cache.deleteEventTicket(e); err != nil {
		return err
	}

	return nil
}

func formatEventSubject(e *event.Notification) string {
	switch {
	case e.Host != "" && e.Service != "":
		return fmt.Sprintf("Host: %v Service: %v is %v", e.Host, e.Service, e.CheckResult.State.String())
	case e.Host != "" && e.Service == "":
		return fmt.Sprintf("Host: %v is %v", e.Host, e.CheckResult.State.String())
	default:
		return fmt.Sprintf("Host: %v Service: %v is %v", e.Host, e.Service, e.CheckResult.State.String())
	}
}

func formatEventComment(e *event.Notification) string {
	if e.CheckResult.Output != "" {
		return fmt.Sprintf("New status: %v Output: %v", e.CheckResult.State.String(), e.CheckResult.Output)
	}

	return e.CheckResult.State.String()
}

func (t *ticketUpdater) comment(e *event.Notification) error {
	_, ticketID, err := t.cache.getEventTicket(e)
	if err != nil {
		return err
	}

	err = t.rtClient.CommentTicket(ticketID, formatEventComment(e))
	if err != nil {
		return err
	}

	if *debug {
		log.Printf("ticket updater: commented ticket #%v", ticketID)
	}

	err = t.cache.updateEventTicket(e, ticketID)
	if err != nil {
		return err
	}

	return nil
}

func (t *ticketUpdater) create(e *event.Notification) error {
	ticket := &rt.Ticket{Queue: t.queue, Subject: formatEventSubject(e), Text: fmt.Sprintf("Output: %s", e.CheckResult.Output)}

	newTicket, err := t.rtClient.NewTicket(ticket)
	if err != nil {
		return err
	}

	if *debug {
		log.Printf("ticket updater: created ticket #%v", newTicket.ID)
	}

	err = t.cache.updateEventTicket(e, newTicket.ID)
	if err != nil {
		return err
	}

	return nil
}

func (t *ticketUpdater) ignore(e *event.Notification) error {
	if *debug {
		log.Printf("ticket updater: ignoring event #%v", formatEventSubject(e))
	}
	return nil
}
