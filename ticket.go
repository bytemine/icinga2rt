package main

import (
	"fmt"
	"log"

	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/rt"
)

type actionFunc func(*ticketUpdater, *event.Notification) error

// condition describes the properties an event must have to match.
type condition struct {
	state    event.State
	oldState event.State
	// existing bool
	owned bool
}

// mapping describes how an event matching condition should be acted upon.
type mapping struct {
	condition condition
	action    actionFunc
}

type ticketUpdater struct {
	cache        *cache
	rtClient     rtClient
	mappings     []mapping
	nobody       string
	queue        string
	closedStatus []string
}

func newTicketUpdater(cache *cache, rtClient rtClient, mappings []mapping, nobody string, queue string, closedStatus []string) *ticketUpdater {
	return &ticketUpdater{cache: cache, rtClient: rtClient, mappings: mappings, nobody: nobody, queue: queue, closedStatus: closedStatus}
}

func (t *ticketUpdater) update(e *event.Notification) error {
	if *debug {
		log.Printf("%x ticket updater: new event: %v", eventID(e), formatEventSubject(e))
	}

	// get a possible old event and ticket from the cache
	oldEvent, ticketID, err := t.cache.getEventTicket(e)
	if err != nil {
		return err
	}

	// assume a fresh event
	// existing := false
	owned := false
	oldState := event.State(event.StateNil)

	// use switch here so we can use break
	switch {
	case oldEvent != nil && ticketID != -1: // existing event found
		oldTicket, err := t.rtClient.Ticket(ticketID)
		if err != nil {
			if *debug {
				log.Printf("%x ticket updater: ticket #%v in cache doesn't exist", eventID(e), ticketID)
			}
			break
		}

		// we have an old event
		// existing = true
		oldState = oldEvent.CheckResult.State
		owned = oldTicket.Owner != t.nobody

		// check if the ticket has a status which signals "closed". if yes, set existing to false
		for _, v := range t.closedStatus {
			if oldTicket.Status == v {
				// existing = false
				oldState = event.State(event.StateNil)
				if *debug {
					log.Printf("%x ticket updater: ticket #%v has closed status: %v", eventID(e), ticketID, oldTicket.Status)
				}
			}
		}
	}

	if *debug {
		log.Printf("%x ticket updater: ticket #%v owned: %v", eventID(e), ticketID, owned)
	}

	for _, v := range t.mappings {
		x := condition{
			state:    e.CheckResult.State,
			oldState: oldState,
			// existing: existing,
			owned: owned,
		}

		if *debug {
			log.Printf("%x ticket updater: matching condition: %+v\tevent: %+v", eventID(e), v.condition, x)
		}

		if v.condition == x {
			if *debug {
				log.Printf("%x ticket updater: matched %+v", eventID(e), v.condition)
			}

			err := v.action(t, e)
			return err
		}
	}

	if *debug {
		log.Printf("%x ticket updater: no condition matched", eventID(e))
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
		log.Printf("%x ticket updater: deleted ticket #%v", eventID(e), updatedTicket.ID)
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
		log.Printf("%x ticket updater: commented ticket #%v", eventID(e), ticketID)
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
		log.Printf("%x ticket updater: created ticket #%v", eventID(e), newTicket.ID)
	}

	err = t.cache.updateEventTicket(e, newTicket.ID)
	if err != nil {
		return err
	}

	return nil
}

func (t *ticketUpdater) ignore(e *event.Notification) error {
	if *debug {
		log.Printf("%x ticket updater: ignoring event #%v", eventID(e), formatEventSubject(e))
	}
	return nil
}
