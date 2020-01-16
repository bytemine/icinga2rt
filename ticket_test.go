package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/rt"
)

const testMappingsCSV = `# state, old state, owned, action
# ignore OK events if no old state is known
OK,,false,ignore
# delete ticket if unowned and was WARNING
OK,WARNING,false,delete
# set ticket status if unowned and was CRITICAL or UNKNOWN
OK,CRITICAL,false,status,resolved,true
OK,UNKNOWN,false,status,customstatus,true
# comment ticket if unowned and was WARNING, CRITICAL or UNKNOWN
OK,WARNING,true,comment
OK,CRITICAL,true,comment
OK,UNKNOWN,true,comment
# create tickets for WARNING, CRITICAL or UNKNOWN if not exisiting
WARNING,,false,create
CRITICAL,,false,create
UNKNOWN,,false,create
# ignore if state hasn't changed
WARNING,WARNING,false,ignore
WARNING,WARNING,true,ignore
CRITICAL,CRITICAL,false,ignore
CRITICAL,CRITICAL,true,ignore
UNKNOWN,UNKNOWN,false,ignore
UNKNOWN,UNKNOWN,true,ignore
# comment tickets on state changes
WARNING,CRITICAL,false,comment
WARNING,CRITICAL,true,comment
WARNING,UNKNOWN,false,comment
WARNING,UNKNOWN,true,comment
CRITICAL,WARNING,false,comment
CRITICAL,WARNING,true,comment
CRITICAL,UNKNOWN,false,comment
CRITICAL,UNKNOWN,true,comment
UNKNOWN,WARNING,false,comment
UNKNOWN,WARNING,true,comment
UNKNOWN,CRITICAL,false,comment
UNKNOWN,CRITICAL,true,comment
`

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
	testMappings, err := readMappings(strings.NewReader(testMappingsCSV))
	if err != nil {
		t.Error(err)
	}

	rt := NewDummyRT()
	cache, cachePath, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, cachePath)

	tu := newTicketUpdater(cache, rt, testMappings, "", "Test-Queue", []string{"deleted"})

	for _, v := range tests {
		t.Logf("%+v", v)
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
