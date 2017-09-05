package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"log"

	"github.com/boltdb/bolt"
	"github.com/bytemine/go-icinga2/event"
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

func (c *cache) getEventTicket(e *event.Notification) (*eventTicket, error) {
	if *debug {
		log.Printf("cache: %v/%v get", e.Host, e.Service)
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
		return nil, err
	}

	return et, nil
}

func (c *cache) updateEventTicket(e *event.Notification, ticketID int) error {
	if *debug {
		log.Printf("cache: %v/%v update", e.Host, e.Service)
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
		log.Printf("cache: %v/%v delete", e.Host, e.Service)
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

func (c *cache) dump() ([]byte, error) {
	var buf bytes.Buffer

	err := c.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(eventBucketName))
		if b == nil {
			return fmt.Errorf("event bucket doesn't exist")
		}

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			et, err := decodeEventTicket(v)
			if err != nil {
				return err
			}

			fmt.Fprintf(&buf, "%v,%v,%v,%v\n", et.TicketID, et.Event.Host, et.Event.Service, et.Event.CheckResult.State)
		}

		return nil
	})

	return buf.Bytes(), err
}

func (c *cache) clean() error {
	okEvents := [][]byte{}

	err := c.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(eventBucketName))
		if b == nil {
			return fmt.Errorf("event bucket doesn't exist")
		}

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			et, err := decodeEventTicket(v)
			if err != nil {
				return err
			}

			if et.Event.CheckResult.State == event.StateOK {
				okEvents = append(okEvents, k)
			}

		}

		return nil
	})

	err = c.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(eventBucketName))
		if b == nil {
			return fmt.Errorf("event bucket doesn't exist")
		}

		for _, v := range okEvents {
			err := b.Delete(v)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
