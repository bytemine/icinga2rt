package main

import (
	"fmt"
	"testing"

	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/rt"
)

var testMappings = []Mapping{
	{
		condition: Condition{
			state:    event.StateCritical,
			existing: false,
			owned:    false,
		},
		action:     (*ticketUpdater).create,
		actionName: "create",
	},
	{
		condition: Condition{
			state:    event.StateCritical,
			existing: true,
			owned:    false,
		},
		action:     (*ticketUpdater).comment,
		actionName: "comment",
	},
	{
		condition: Condition{
			state:    event.StateWarning,
			existing: false,
			owned:    false,
		},
		action:     (*ticketUpdater).create,
		actionName: "create",
	},
	{
		condition: Condition{
			state:    event.StateWarning,
			existing: true,
			owned:    false,
		},
		action:     (*ticketUpdater).comment,
		actionName: "comment",
	},
	{
		condition: Condition{
			state:    event.StateOK,
			existing: true,
			owned:    false,
		},
		action:     (*ticketUpdater).delete,
		actionName: "delete",
	},
}

// the order is important.
// we can't check everything here. it isn't checked if the comments are really attached to a ticket, or the status of a ticket,
// as that would require a complete RT-mock. maybe using an interface would be good for the rt client.
var tests = []struct {
	Event        *event.Notification
	ExistsBefore bool // exists in cache before processing, checks event and ticket id
	ExistsAfter  bool // exists in cache after processing, checks event and ticket id
}{
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateWarning,
			},
		},
		ExistsBefore: false,
		ExistsAfter:  true,
	},
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateCritical,
			},
		},
		ExistsBefore: true,
		ExistsAfter:  true,
	},
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateOK,
			},
		},
		ExistsBefore: true,
		ExistsAfter:  false,
	},
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateOK,
			},
		},
		ExistsBefore: false,
		ExistsAfter:  false,
	},
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateCritical,
			},
		},
		ExistsBefore: false,
		ExistsAfter:  true,
	},
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateWarning,
			},
		},
		ExistsBefore: true,
		ExistsAfter:  true,
	},
	{
		Event: &event.Notification{
			Host:    "example.com",
			Service: "example",
			CheckResult: event.CheckResultData{
				State: event.StateOK,
			},
		},
		ExistsBefore: true,
		ExistsAfter:  false,
	},
}

func TestTicketUpdaterUpdate(t *testing.T) {
	rt := NewDummyRT()
	cache, cachePath, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, cachePath)

	tu := newTicketUpdater(cache, rt, testMappings, "", "Test-Queue", []string{"deleted"})

	for _, v := range tests {
		if v.ExistsBefore {
			x, ticketID, err := cache.getEventTicket(v.Event)
			if err != nil {
				t.Error(err)
			}

			if x == nil {
				t.Log("before: event in cache is nil")
				t.Fail()
			}

			if ticketID == -1 {
				t.Log("before: ticket id is nil")
				t.Fail()
			}
		}

		err := tu.update(v.Event)
		if err != nil {
			t.Error(err)
		}

		if v.ExistsAfter {
			x, ticketID, err := cache.getEventTicket(v.Event)
			if err != nil {
				t.Error(err)
			}

			if x == nil {
				t.Log("after: event in cache is nil")
				t.Fail()
			}

			if ticketID == -1 {
				t.Log("after: ticket id is nil")
				t.Fail()
			}

			if v.Event.CheckResult.State != x.CheckResult.State {
				t.Logf("after: event state: %v expected: %v", x.CheckResult.State, v.Event.CheckResult.State)
				t.Fail()
			}
		}
	}
}

// DummyClient is a mock RT client used for testing.
type DummyRT struct {
	tickets []rt.Ticket
}

func NewDummyRT() *DummyRT {
	return &DummyRT{tickets: make([]rt.Ticket, 0)}
}

func (d *DummyRT) Ticket(id int) (*rt.Ticket, error) {
	if len(d.tickets) > id {
		return &d.tickets[id], nil
	}
	return nil, fmt.Errorf("no ticket")
}

func (d *DummyRT) NewTicket(ticket *rt.Ticket) (*rt.Ticket, error) {
	ticket.ID = len(d.tickets)
	d.tickets = append(d.tickets, *ticket)
	return ticket, nil
}

func (d *DummyRT) UpdateTicket(ticket *rt.Ticket) (*rt.Ticket, error) {
	d.tickets[ticket.ID] = *ticket
	return ticket, nil
}

func (d *DummyRT) CommentTicket(id int, comment string) error {
	return nil
}
