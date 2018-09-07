package rt

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

type Ticket struct {
	ID              int
	Queue           string
	Owner           string
	Creator         string
	Subject         string
	Status          string
	Priority        string
	InitialPriority string
	FinalPriority   string
	Requestors      string
	Cc              string
	AdminCc         string
	Created         string
	Starts          string
	Started         string
	Due             string
	Resolved        string
	Told            string
	LastUpdated     string
	TimeEstimated   string
	TimeWorked      string
	TimeLeft        string
	Text            string
}

// This ignores the Text of a ticket for now.
func (t *Ticket) decode(r io.Reader) error {
	s := bufio.NewScanner(r)

	for s.Scan() {
		if strings.HasPrefix(s.Text(), "# Ticket ") {
			return fmt.Errorf("ticket doesn't exist")
		}

		if !strings.Contains(s.Text(), ": ") {
			continue
		}

		fs := strings.Split(s.Text(), ": ")
		if len(fs) < 2 {
			continue
		}

		c := fs[1]

		switch fs[0] {
		case "id":
			idStr := strings.TrimPrefix(c, "ticket/")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			t.ID = id
		case "Queue":
			t.Queue = c
		case "Owner":
			t.Owner = c
		case "Creator":
			t.Creator = c
		case "Subject":
			t.Subject = c
		case "Status":
			t.Status = c
		case "Priority":
			t.Priority = c
		case "FinalPriority":
			t.FinalPriority = c
		case "Requestors":
			t.Requestors = c
		case "Cc":
			t.Cc = c
		case "AdminCc":
			t.AdminCc = c
		case "Created":
			t.Created = c
		case "Starts":
			t.Starts = c
		case "Started":
			t.Started = c
		case "Due":
			t.Due = c
		case "Resolved":
			t.Resolved = c
		case "Told":
			t.Told = c
		case "LastUpdated":
			t.LastUpdated = c
		case "TimeEstimated":
			t.TimeEstimated = c
		case "TimeWorked":
			t.TimeWorked = c
		case "TimeLeft":
			t.TimeLeft = c
		}
	}
	return nil
}

func (t *Ticket) encode() string {
	out := []string{}

	if t.ID == 0 {
		out = append(out, "id: new")
	} else {
		out = append(out, fmt.Sprintf("id: %v", t.ID))
	}

	if t.Queue != "" {
		out = append(out, fmt.Sprintf("Queue: %s", t.Queue))
	}

	if t.Owner != "" {
		out = append(out, fmt.Sprintf("Owner: %s", t.Owner))
	}

	if t.Subject != "" {
		out = append(out, fmt.Sprintf("Subject: %s", t.Subject))
	}

	if t.Status != "" {
		out = append(out, fmt.Sprintf("Status: %s", t.Status))
	}

	if t.Priority != "" {
		out = append(out, fmt.Sprintf("Priority: %s", t.Priority))
	}

	if t.FinalPriority != "" {
		out = append(out, fmt.Sprintf("FinalPriority: %s", t.FinalPriority))
	}

	if t.Requestors != "" {
		out = append(out, fmt.Sprintf("Requestors: %s", t.Requestors))
	}

	if t.Cc != "" {
		out = append(out, fmt.Sprintf("Cc: %s", t.Cc))
	}

	if t.AdminCc != "" {
		out = append(out, fmt.Sprintf("AdminCc: %s", t.AdminCc))
	}

	if t.Starts != "" {
		out = append(out, fmt.Sprintf("Starts: %s", t.Starts))
	}

	if t.Started != "" {
		out = append(out, fmt.Sprintf("Started: %s", t.Started))
	}

	if t.Due != "" {
		out = append(out, fmt.Sprintf("Due: %s", t.Due))
	}

	if t.Resolved != "" {
		out = append(out, fmt.Sprintf("Resolved: %s", t.Resolved))
	}

	if t.TimeEstimated != "" {
		out = append(out, fmt.Sprintf("TimeEstimated: %s", t.TimeEstimated))
	}

	if t.TimeWorked != "" {
		out = append(out, fmt.Sprintf("TimeWorked: %s", t.TimeWorked))
	}

	if t.TimeLeft != "" {
		out = append(out, fmt.Sprintf("TimeLeft: %s", t.TimeLeft))
	}

	if t.Text != "" {
		out = append(out, fmt.Sprintf("Text: %s", t.Text))
	}

	return strings.Join(out, "\n")
}

// Client is a RT REST 1.0 client.
type Client struct {
	url                *url.URL
	user               string
	password           string
	insecureSkipVerify bool
	dummy              bool
	dummyTickets       []Ticket
}

// NewClient prepares a Client for usage.
func NewClient(rtURL string, user, password string, insecureSkipVerify bool) (*Client, error) {
	x, err := url.Parse(rtURL)
	if err != nil {
		return nil, err
	}
	return &Client{url: x, user: user, password: password, insecureSkipVerify: insecureSkipVerify}, nil
}

func NewDummyClient() *Client {
	return &Client{dummy: true, dummyTickets: make([]Ticket, 0)}
}

func (c *Client) Ticket(id int) (*Ticket, error) {
	if c.dummy {
		if len(c.dummyTickets) > id {
			return &c.dummyTickets[id], nil
		}
		return nil, fmt.Errorf("no ticket")
	}
	x := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecureSkipVerify}}}

	query := url.Values{}
	query.Set("user", c.user)
	query.Set("pass", c.password)

	u := url.URL{Scheme: "https", Host: c.url.Host, Path: filepath.Join(c.url.Path, "REST", "1.0", "ticket", strconv.Itoa(id)), RawQuery: query.Encode()}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := x.Do(req)
	if err != nil {
		return nil, err
	}

	t := &Ticket{}

	err = t.decode(res.Body)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (c *Client) NewTicket(ticket *Ticket) (*Ticket, error) {
	if c.dummy {
		ticket.ID = len(c.dummyTickets)
		c.dummyTickets = append(c.dummyTickets, *ticket)
		return ticket, nil
	}
	x := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecureSkipVerify}}}
	query := url.Values{}
	query.Set("user", c.user)
	query.Set("pass", c.password)

	u := url.URL{Scheme: "https", Host: c.url.Host, Path: filepath.Join(c.url.Path, "REST", "1.0", "ticket", "new"), RawQuery: query.Encode()}

	form := url.Values{}
	form.Add("content", ticket.encode())

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	res, err := x.Do(req)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(res.Body)

	id := 0

	for s.Scan() {
		if strings.HasPrefix(s.Text(), "# Ticket ") {
			fs := strings.Fields(s.Text())
			if len(fs) != 4 {
				return nil, fmt.Errorf("response didn't contain ticket number.")
			}

			id, err = strconv.Atoi(fs[2])
			if err != nil {
				return nil, err
			}
		}
	}

	newTicket, err := c.Ticket(id)
	if err != nil {
		return nil, err
	}

	return newTicket, nil

}

func (c *Client) UpdateTicket(ticket *Ticket) (*Ticket, error) {
	if c.dummy {
		c.dummyTickets[ticket.ID] = *ticket
		return ticket, nil
	}
	x := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecureSkipVerify}}}
	query := url.Values{}
	query.Set("user", c.user)
	query.Set("pass", c.password)

	u := url.URL{Scheme: "https", Host: c.url.Host, Path: filepath.Join(c.url.Path, "REST", "1.0", "ticket", strconv.Itoa(ticket.ID), "edit"), RawQuery: query.Encode()}

	form := url.Values{}
	form.Add("content", ticket.encode())

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	res, err := x.Do(req)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(res.Body)

	id := 0

	for s.Scan() {
		if strings.HasPrefix(s.Text(), "# Ticket ") {
			fs := strings.Fields(s.Text())
			if len(fs) != 4 {
				return nil, fmt.Errorf("response didn't contain ticket number.")
			}

			id, err = strconv.Atoi(fs[2])
			if err != nil {
				return nil, err
			}
		}
	}

	newTicket, err := c.Ticket(id)
	if err != nil {
		return nil, err
	}

	return newTicket, nil

}

func (c *Client) CommentTicket(ticketID int, comment string) error {
	if c.dummy {
		return nil
	}
	x := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecureSkipVerify}}}
	query := url.Values{}
	query.Set("user", c.user)
	query.Set("pass", c.password)

	u := url.URL{Scheme: "https", Host: c.url.Host, Path: filepath.Join(c.url.Path, "REST", "1.0", "ticket", strconv.Itoa(ticketID), "comment"), RawQuery: query.Encode()}

	form := url.Values{}
	form.Add("content", fmt.Sprintf("id: %v\nAction: comment\nText: %v", ticketID, comment))

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	res, err := x.Do(req)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(res.Body)

	id := 0

	for s.Scan() {
		if strings.HasPrefix(s.Text(), "# Ticket ") {
			fs := strings.Fields(s.Text())
			if len(fs) != 4 {
				return fmt.Errorf("response didn't contain ticket number.")
			}

			id, err = strconv.Atoi(fs[2])
			if err != nil {
				return err
			}
		}
	}

	id = id

	return nil

}
