package main

import (
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
		action: (*ticketUpdater).create,
	},
	{
		condition: Condition{
			state:    event.StateCritical,
			existing: true,
			owned:    false,
		},
		action: (*ticketUpdater).comment,
	},
	{
		condition: Condition{
			state:    event.StateWarning,
			existing: false,
			owned:    false,
		},
		action: (*ticketUpdater).create,
	},
	{
		condition: Condition{
			state:    event.StateWarning,
			existing: true,
			owned:    false,
		},
		action: (*ticketUpdater).comment,
	},
	{
		condition: Condition{
			state:    event.StateOK,
			existing: true,
			owned:    false,
		},
		action: (*ticketUpdater).delete,
	},
}

// the order is important.
// we can't check everything here. it isn't checked if the comments are really attached to a ticket,
// as that would require a complete RT-mock.
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
	rt := rt.NewDummyClient()
	cache, cachePath, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, cachePath)

	tu := newTicketUpdater(cache, rt, defaultConfig.Ticket.Mappings, "Nobody", "Test-Queue")

	for _, v := range tests {
		t.Logf("%#v", v)

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
				t.Fail()
			}
		}
	}
}
