package filter

import (
	"github.com/bytemine/go-icinga2/event"
)

// Matcher implements rules for matching sets of filters.
type Matcher interface {
	// Match the event according to the rules of the filter set.
	// Returns true on match, false otherwise.
	Match(n event.Notification) bool
}

// Filter usable as filter.
type Filter struct {
	Host             string                 `json:",omitempty"`
	Service          string                 `json:",omitempty"`
	Users            []string               `json:",omitempty"`
	Author           string                 `json:",omitempty"`
	Text             string                 `json:",omitempty"`
	NotificationType event.NotificationType `json:"notification_type,omitempty"`
}

// Any matcher matches like a disjunctive normal form.
//
// Each filter is evaluated using filter.All, if any filter matches, the filter set matches.
type Any []Filter

// Match implementation.
func (a Any) Match(n event.Notification) bool {
	for _, f := range a {
		if f.All(n) {
			return true
		}
	}

	return false
}

// All matchers match like a conjunctive normal form.
//
// Each filter is evaluated using filter.Any, if all filters match, the filter set matches.
type All []Filter

// Match implementation.
func (a All) Match(n event.Notification) bool {
	for _, f := range a {
		if !f.Any(n) {
			return false
		}
	}

	return true
}

// containsStrings tests if two slices contain the same strings, used to compare users.
func containsAll(a, b []string) bool {
	m := make(map[string]struct{})
	for _, v := range a {
		m[v] = struct{}{}
	}

	for _, v := range b {
		if _, ok := m[v]; !ok {
			return false
		}
	}

	return true
}

// containsAny tests if two slices have common strings, used to compare users.
func containsAny(a, b []string) bool {
	for _, v := range a {
		for _, w := range b {
			if v == w {
				return true
			}
		}
	}

	return false
}

// All returns true if all values of the receiver are contained in x.
// Values not set in the receiver aren't considered.
// The comparison is shallow, contents of CheckResult aren't considered.
func (f Filter) All(x event.Notification) bool {
	if f.Author != "" && f.Author != x.Author {
		return false
	}

	if f.Host != "" && f.Host != x.Host {
		return false
	}

	if f.NotificationType != "" && f.NotificationType != x.NotificationType {
		return false
	}

	if f.Service != "" && f.Service != x.Service {
		return false
	}

	if f.Text != "" && f.Text != x.Text {
		return false
	}

	if len(f.Users) != 0 && !containsAll(f.Users, x.Users) {
		return false
	}

	return true
}

// Any returns true if one of the values of the receiver is contained in x.
// Values not set in the receiver aren't considered.
// The comparison is shallow, contents of CheckResult aren't considered.
func (f Filter) Any(x event.Notification) bool {
	// Return true if the filter is empty.
	if f.Author == "" &&
		f.Host == "" &&
		f.NotificationType == "" &&
		f.Service == "" &&
		f.Text == "" &&
		len(f.Users) == 0 {
		return true
	}

	if f.Author != "" && f.Author == x.Author {
		return true
	}

	if f.Host != "" && f.Host == x.Host {
		return true
	}

	if f.NotificationType != "" && f.NotificationType == x.NotificationType {
		return true
	}

	if f.Service != "" && f.Service == x.Service {
		return true
	}

	if f.Text != "" && f.Text == x.Text {
		return true
	}

	if len(f.Users) != 0 && containsAny(f.Users, x.Users) {
		return true
	}

	return false
}
