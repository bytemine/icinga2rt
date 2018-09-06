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

type Condition struct {
	State    string
	state    event.State
	Existing bool
	Owned    bool
}

func (c *Condition) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, c)
	if err != nil {
		return err
	}

	switch c.State {
	case "OK":
		c.state = event.StateOK
	case "WARNING":
		c.state = event.StateWarning
	case "CRITICAL":
		c.state = event.StateCritical
	case "UNKNOWN":
		c.state = event.StateUnknown
	default:
		return fmt.Errorf("unknown state: %v", c.State)
	}

	return nil
}

func (c *Condition) MarshalJSON() ([]byte, error) {
	c.State = c.state.String()
	if c.State == "" {
		return nil, fmt.Errorf("unknown state: %v", float64(c.state))
	}

	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Mapping struct {
	Condition Condition
	Action    string
	action    actionFunc // hidden field, is filled by custom unmarshalling function
}

func (m *Mapping) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, m)
	if err != nil {
		return err
	}

	switch m.Action {
	case actionStringDelete:
		m.action = (*ticketUpdater).delete
	case actionStringComment:
		m.action = (*ticketUpdater).comment
	case actionStringCreate:
		m.action = (*ticketUpdater).create
	case actionStringIgnore:
		m.action = (*ticketUpdater).ignore
	default:
		return fmt.Errorf("unknown action: %v", m.Action)
	}

	return nil
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
			State:    v.Condition.State,
			Existing: existing,
			Owned:    owned,
		}

		if *debug {
			log.Printf("ticket updater: matching\nCondition: %#v\nEvent:     %#v", v.Condition, x)
		}

		if v.Condition == x {
			if *debug {
				log.Printf("ticket updater: matched %v", v.Action)
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
