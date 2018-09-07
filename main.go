package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/bytemine/go-icinga2"
	"github.com/bytemine/go-icinga2/event"
	"github.com/bytemine/icinga2rt/rt"
)

const version = "0.0.12"
const icingaQueueName = "icinga2rt"

var writeExample = flag.Bool("example", false, "write example configuration file as icinga2rt.json.example to current directory")
var configFile = flag.String("config", "/etc/bytemine/icinga2rt.json", "configuration file")
var debug = flag.Bool("debug", false, "debug mode, print log messages")
var debugEvents = flag.Bool("debugevents", false, "print received events")
var showVersion = flag.Bool("version", false, "display version and exit")

// openEventStreamer connects to the icinga2 API, exponentially backing off when the connection fails
func openEventStreamer(retries int, icingaClient *icinga2.Client, queue string, filter string, streamtype ...event.StreamType) (io.Reader, error) {
	exp := uint(0)

	var err error
	for tries := 0; tries < retries; tries++ {
		if *debug {
			log.Printf("main: connecting to icinga, try: %v", tries+1)
		}

		var r io.Reader
		r, err = icingaClient.EventStream(queue, filter, streamtype...)
		if err != nil {
			if *debug {
				log.Printf("main: couldn't connect to icinga: %v", err)
				log.Printf("main: waiting %v seconds before trying again.", 1<<exp)
			}
			time.Sleep(time.Duration(1<<exp) * time.Second)
			exp++
			continue
		}

		return r, nil
	}

	return nil, err
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *writeExample {
		err := writeConfig("icinga2rt.json.example", &defaultConfig)
		if err != nil {
			log.Fatal("FATAL: init:", err)
		}
		os.Exit(0)
	}

	conf, err := readConfig(*configFile)
	if err != nil {
		log.Fatalf("FATAL: init: Couldn't open config file %v: %v", *configFile, err)
	}

	if err := checkConfig(conf); err != nil {
		log.Fatal("FATAL: init:", err)
	}

	icingaClient, err := icinga2.NewClient(conf.Icinga.URL, conf.Icinga.User, conf.Icinga.Password, conf.Icinga.Insecure)
	if err != nil {
		log.Fatal("FATAL: init:", err)
	}

	r, err := openEventStreamer(conf.Icinga.Retries, icingaClient, icingaQueueName, "", event.StreamTypeNotification)
	if err != nil {
		log.Fatal("FATAL: init:", err)
	}

	eventCache, err := openCache(conf.Cache.File)
	if err != nil {
		log.Fatal("FATAL: init:", err)
	}

	// we may want to have no rtClient for debugging.
	var rtClient *rt.Client
	if conf.RT != (rtConfig{}) {
		rtClient, err = rt.NewClient(conf.RT.URL, conf.RT.User, conf.RT.Password, conf.RT.Insecure)
		if err != nil {
			log.Fatal("FATAL: init:", err)
		}
	} else {
		if *debug {
			log.Println("init: using dummy rt client")
		}
		rtClient = rt.NewDummyClient()
	}

	tu := newTicketUpdater(eventCache, rtClient, conf.Ticket.Mappings, conf.Ticket.Nobody, conf.Ticket.Queue)

	dec := json.NewDecoder(r)
	for {
		var x event.Notification

		err := dec.Decode(&x)
		if err != nil {
			if *debug {
				log.Printf("main: %v", err)
				log.Printf("main: trying to reconnect to icinga.")
			}

			r, err := openEventStreamer(conf.Icinga.Retries, icingaClient, icingaQueueName, "", event.StreamTypeNotification)
			if err != nil {
				log.Fatal("FATAL: main:", err)
			}

			dec = json.NewDecoder(r)
			continue
		}

		if *debug && *debugEvents {
			buf, err := json.Marshal(x)
			if err != nil {
				log.Fatal("FATAL: main:", err)
			}
			log.Println("main: event stream:", string(buf))
		}

		err = tu.update(&x)
		if err != nil {
			log.Fatal("FATAL: main:", err)
		}
	}
}
