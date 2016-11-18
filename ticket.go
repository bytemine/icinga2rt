package main

import (
	"fmt"
	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/rt"
	"log"
)

type permitFunc func(e *event.Notification) bool

func newPermitFunc(permit []event.State) permitFunc {
	return func(e *event.Notification) bool {
		for _, x := range permit {
			if e.CheckResult.State == x {
				return true
			}
		}
		return false
	}
}

type eventTicket struct {
	Event    *event.Notification
	TicketID int
}

type ticketUpdater struct {
	cache    *cache
	rtClient *rt.Client
	pf       permitFunc
	nobody   string
	queue    string
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

func newTicketUpdater(cache *cache, rtClient *rt.Client, pf permitFunc, nobody string, queue string) *ticketUpdater {
	return &ticketUpdater{cache: cache, rtClient: rtClient, pf: pf, nobody: nobody, queue: queue}
}

func (t *ticketUpdater) updateTicket(e *event.Notification) error {
	et, err := t.cache.getEventTicket(e)
	if err != nil {
		return err
	}

	oldState := event.StateUnknown
	if et != nil {
		oldState = et.Event.CheckResult.State
	}

	if *debug {
		log.Printf("ticket updater: incoming event %v/%v %v (was %v)\n", e.Host, e.Service, e.CheckResult.State.String(), oldState.String())
	}

	if !t.pf(e) {
		log.Printf("ticket updater: ignoring event for %v/%v %v due to permit function.", e.Host, e.Service, e.CheckResult.State.String())
		return nil
	}

	if et == nil {
		return t.newEvent(e)
	}

	return t.oldEvent(e, et.Event, et.TicketID)
}

func (t *ticketUpdater) newEvent(e *event.Notification) error {
	if e.CheckResult.State == event.StateOK {
		return nil
	}

	return t.createTicket(e)
}

func (t *ticketUpdater) oldEvent(newEvent *event.Notification, oldEvent *event.Notification, ticketID int) error {
	switch newEvent.CheckResult.State {
	case event.StateOK:
		return t.deleteOrCommentTicket(newEvent, oldEvent, ticketID)
	default:
		// don't update if the state hasn't changed
		if newEvent.CheckResult.State != oldEvent.CheckResult.State {
			return t.commentTicket(newEvent, oldEvent, ticketID)
		}
		return nil
	}
}

func (t *ticketUpdater) createTicket(e *event.Notification) error {
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

func (t *ticketUpdater) deleteOrCommentTicket(newEvent, oldEvent *event.Notification, ticketID int) error {
	oldTicket, err := t.rtClient.Ticket(ticketID)
	if err != nil {
		return err
	}

	switch oldTicket.Owner {
	case t.nobody:
		return t.deleteTicket(newEvent, oldEvent, ticketID)
	default:
		return t.commentTicket(newEvent, oldEvent, ticketID)
	}
}

func (t *ticketUpdater) commentTicket(newEvent, oldEvent *event.Notification, ticketID int) error {
	// Comment existing ticket with new status.
	err := t.rtClient.CommentTicket(ticketID, formatEventComment(newEvent))
	if err != nil {
		return err
	}

	if *debug {
		log.Printf("ticket updater: commented ticket #%v", ticketID)
	}

	err = t.cache.updateEventTicket(newEvent, ticketID)
	if err != nil {
		return err
	}

	return nil
}

func (t *ticketUpdater) deleteTicket(newEvent, oldEvent *event.Notification, ticketID int) error {
	newTicket := &rt.Ticket{ID: ticketID, Status: "deleted"}

	updatedTicket, err := t.rtClient.UpdateTicket(newTicket)
	if err != nil {
		return err
	}

	if *debug {
		log.Printf("ticket updater: deleted ticket #%v", updatedTicket.ID)
	}

	if err = t.cache.deleteEventTicket(newEvent); err != nil {
		return err
	}

	return nil
}
