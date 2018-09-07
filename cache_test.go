package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytemine/go-icinga2/event"
)

var testEvent = &event.Notification{Host: "example.com", Service: "example"}

func TestEventID(t *testing.T) {
	if !bytes.Equal([]byte{0x6b, 0x19, 0x7e, 0xf, 0xbc, 0x99, 0x88, 0xa8}, eventID(testEvent)) {
		t.Fail()
	}
}

func TestEncodeDecode(t *testing.T) {
	et := eventTicket{Event: testEvent, TicketID: 1234}

	buf, err := encodeEventTicket(&et)
	if err != nil {
		t.Error(err)
	}

	xEt, err := decodeEventTicket(buf)
	if err != nil {
		t.Error(err)
	}

	if et.Event.Host != xEt.Event.Host || et.Event.Service != xEt.Event.Service {
		t.Fail()
	}

	if et.TicketID != xEt.TicketID {
		t.Fail()
	}
}

func tempCache() (*cache, string, error) {
	path, err := ioutil.TempDir("", "icinga2rt")
	if err != nil {
		return nil, "", err
	}

	path = filepath.Join(path, "icinga2rt.bolt")
	c, err := openCache(path)
	return c, path, err
}

func removeCache(cache *cache, path string) error {
	err := cache.Close()
	if err != nil {
		return err
	}

	if path == "" {
		return fmt.Errorf("path is empty")
	}

	err = os.Remove(path)
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Dir(path))
	return err
}

func TestGetEventTicket(t *testing.T) {
	cache, path, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, path)

	err = cache.updateEventTicket(testEvent, 1234)
	if err != nil {
		t.Error()
	}

	e, ticketId, err := cache.getEventTicket(testEvent)
	if err != nil {
		t.Error(err)
	}

	if e.Host != testEvent.Host || e.Service != testEvent.Service || ticketId != 1234 {
		t.Fail()
	}
}

func TestGetNotExistingEventTicket(t *testing.T) {
	cache, path, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, path)

	e, ticketId, err := cache.getEventTicket(testEvent)
	if err != nil {
		t.Error(err)
	}

	if e != nil || ticketId != -1 {
		t.Fail()
	}
}

func TestUpdateEventTicket(t *testing.T) {
	cache, path, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, path)

	err = cache.updateEventTicket(testEvent, 1234)
	if err != nil {
		t.Error()
	}

	e, ticketId, err := cache.getEventTicket(testEvent)
	if err != nil {
		t.Error(err)
	}

	if e.Host != testEvent.Host || e.Service != testEvent.Service || ticketId != 1234 {
		t.Fail()
	}

	err = cache.updateEventTicket(testEvent, 4321)
	if err != nil {
		t.Error()
	}

	e, ticketId, err = cache.getEventTicket(testEvent)
	if err != nil {
		t.Error(err)
	}

	if e.Host != testEvent.Host || e.Service != testEvent.Service || ticketId != 4321 {
		t.Fail()
	}
}

func TestDeleteEventTicket(t *testing.T) {
	cache, path, err := tempCache()
	if err != nil {
		t.Error(err)
	}
	defer removeCache(cache, path)

	err = cache.updateEventTicket(testEvent, 1234)
	if err != nil {
		t.Error()
	}

	err = cache.deleteEventTicket(testEvent)
	if err != nil {
		t.Error()
	}

	e, ticketId, err := cache.getEventTicket(testEvent)
	if err != nil {
		t.Error(err)
	}

	if e != nil || ticketId != -1 {
		t.Fail()
	}
}
