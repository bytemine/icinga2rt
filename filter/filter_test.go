package filter

import (
	"testing"

	"github.com/bytemine/go-icinga2/event"
)

var tests = []struct {
	f   Filter
	x   event.Notification
	Any bool
	All bool
}{
	{ // empty filter should match empty notification
		Filter{},
		event.Notification{},
		true,
		true,
	},
	{ // empty filter should match any notification
		Filter{},
		event.Notification{
			Host:             "test",
			Service:          "test",
			Users:            []string{"james", "tiberius", "kirk"},
			Author:           "test",
			Text:             "test",
			NotificationType: event.NotificationProblem,
		},
		true,
		true,
	},
	{ // filters should match exact matches
		Filter{
			Host:             "test",
			Service:          "test",
			Users:            []string{"james", "tiberius", "kirk"},
			Author:           "test",
			Text:             "test",
			NotificationType: event.NotificationProblem,
		},
		event.Notification{
			Host:             "test",
			Service:          "test",
			Users:            []string{"james", "tiberius", "kirk"},
			Author:           "test",
			Text:             "test",
			NotificationType: event.NotificationProblem,
		},
		true,
		true,
	},
	{ // empty filter fields should be ignored in comparisons
		Filter{
			Users: []string{"james", "tiberius", "kirk"},
		},
		event.Notification{
			Host:             "test",
			Service:          "test",
			Users:            []string{"james", "tiberius", "kirk"},
			Author:           "test",
			Text:             "test",
			NotificationType: event.NotificationProblem,
		},
		true,
		true,
	},
	{ // any filter should match, all filter should fail
		Filter{
			Users: []string{"tiberius"},
		},
		event.Notification{
			Host:             "test",
			Service:          "test",
			Users:            []string{"james", "tiberius", "kirk"},
			Author:           "test",
			Text:             "test",
			NotificationType: event.NotificationProblem,
		},
		true,
		false,
	},
	{ // any filter should fail, all filter should fail
		Filter{
			Users: []string{"picard"},
		},
		event.Notification{
			Host:             "test",
			Service:          "test",
			Users:            []string{"james", "tiberius", "kirk"},
			Author:           "test",
			Text:             "test",
			NotificationType: event.NotificationProblem,
		},
		false,
		false,
	},
}

func TestFilters(t *testing.T) {
	for _, v := range tests {
		if v.f.All(v.x) != v.All {
			t.Fail()
		}

		if v.f.Any(v.x) != v.Any {
			t.Fail()
		}
	}
}
