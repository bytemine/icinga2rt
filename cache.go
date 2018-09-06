package main

import (
	"bytes"
	"encoding/gob"
	"hash/fnv"
	"log"

	"github.com/bytemine/go-icinga2/event"
	bolt "github.com/etcd-io/bbolt" // bbolt is the continuation of bolt and for now is usable as drop in replacement
)

const eventBucketName = "events"
const pendingBucketName = "pendingEvents"

// eventID generates an internal id to prevent using nested maps
func eventID(e *event.Notification) []byte {
	h := fnv.New64a()

	// the fvn hash always returns nil error, so we can ignore it here
	h.Write([]byte(e.Host))
	h.Write([]byte(e.Service))

	return h.Sum(nil)
}

type cache struct {
	*bolt.DB
	debug bool
}

func openCache(path string) (*cache, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	return &cache{DB: db}, nil
}

// eventTicket is a helper struct for saving to bolt
type eventTicket struct {
	Event    *event.Notification
	TicketID int
}

func decodeEventTicket(x []byte) (*eventTicket, error) {
	var et eventTicket
	buf := bytes.NewBuffer(x)
	d := gob.NewDecoder(buf)

	if err := d.Decode(&et); err != nil {
		return nil, err
	}

	return &et, nil
}

func encodeEventTicket(et *eventTicket) ([]byte, error) {
	var x bytes.Buffer

	e := gob.NewEncoder(&x)
	err := e.Encode(et)
	if err != nil {
		return nil, err
	}

	return x.Bytes(), nil
}

func (c *cache) getEventTicket(e *event.Notification) (*event.Notification, int, error) {
	if *debug {
		log.Printf("cache: get event for %v/%v", e.Host, e.Service)
	}

	eID := eventID(e)

	var et *eventTicket
	err := c.DB.View(func(tx *bolt.Tx) error {
		var err error // declare it here so we can use = instead of := to prevent shadowing
		eventBucket := tx.Bucket([]byte(eventBucketName))
		if eventBucket == nil {
			return nil
		}

		x := eventBucket.Get(eID)
		// if we don't have a saved event just return nil
		if x == nil {
			return nil
		}

		et, err = decodeEventTicket(x)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, -1, err
	}

	return et.Event, et.TicketID, nil
}

func (c *cache) updateEventTicket(e *event.Notification, ticketID int) error {
	if *debug {
		log.Printf("cache: update event for %v/%v", e.Host, e.Service)
	}

	eID := eventID(e)

	err := c.DB.Update(func(tx *bolt.Tx) error {
		hostBucket, err := tx.CreateBucketIfNotExists([]byte(eventBucketName))
		if err != nil {
			return err
		}

		x, err := encodeEventTicket(&eventTicket{Event: e, TicketID: ticketID})
		if err != nil {
			return err
		}

		return hostBucket.Put(eID, x)
	})

	return err
}

func (c *cache) deleteEventTicket(e *event.Notification) error {
	if *debug {
		log.Printf("cache: delete event for %v/%v", e.Host, e.Service)
	}

	eID := eventID(e)

	err := c.DB.Update(func(tx *bolt.Tx) error {
		hostBucket, err := tx.CreateBucketIfNotExists([]byte(eventBucketName))
		if err != nil {
			return err
		}

		return hostBucket.Delete(eID)
	})

	return err
}
