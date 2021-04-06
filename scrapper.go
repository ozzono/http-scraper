// Copyright 2015 Ronoaldo JLP <ronoaldo@gmail.com>
// Licensed under the Apache License, Version 2.0

package scraper

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

var (
	// ErrTooManyRedirects is returned when the scraper reaches more than 10 redirects.
	ErrTooManyRedirects = errors.New("scraper: too many redirects")
)

// Scraper implements a statefull HTTP client for interacting with websites.
type Scraper struct {
	b     string
	j     *cookiejar.Jar
	c     *http.Client
	debug bool

	// lastURL records the last seen URL using the CheckRedirect function.
	// TODO(ronoaldo): change to a history of recent URLs.
	history *History
}

// New initializes a new Scraper with an in-memory cookie management.
func New() *Scraper {
	return ReuseClient(http.DefaultClient)
}

// CustomNew initializes a new Bot with an in-memory cookie management and with custom http.Client.
func CustomNew(c *http.Client) *Scraper {
	return ReuseClient(c)
}

func ReuseClient(c *http.Client) *Scraper {
	jar, err := cookiejar.New(nil)
	if err != nil {
		// Currently, cookiejar.Nil never returns an error
		panic(err)
	}
	c.Jar = jar
	scraper := &Scraper{
		j:       jar,
		c:       c,
		history: &History{},
	}
	origTransport := c.Transport
	if origTransport == nil {
		origTransport = http.DefaultTransport
	}
	t := &transport{
		t: origTransport,
		b: scraper,
	}
	scraper.c.Transport = t
	scraper.c.CheckRedirect = scraper.checkRedirect
	return scraper
}

// Do sends the HTTP request using the http.Client.Do.
// It returns a nil page if there is a network error.
// It will also return an error if the response is not 2xx,
// but the returned page is non-nil, and you can parse the error body.
func (scraper *Scraper) Do(req *http.Request) (*Page, error) {
	scraper.history.Add(scraper.b + req.URL.String())
	resp, err := scraper.c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("scraper: non 2xx response code: %d: %s", resp.StatusCode, resp.Status)
	}
	return &Page{resp: resp}, nil
}

// GET performs the HTTP GET to the provided URL and returns a Page.
// It returns a nil page if there is a network error.
// It will also return an error if the response is not 2xx,
// but the returned page is non-nil, and you can parse the error body.
func (scraper *Scraper) GET(url string) (*Page, error) {
	scraper.history.Add(scraper.b + url)
	resp, err := scraper.c.Get(scraper.b + url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("scraper: non 2xx response code: %d: %s", resp.StatusCode, resp.Status)
	}
	return &Page{resp: resp}, nil
}

// POST performs an HTTP POST to the provided URL,
// using the form as a payload, and returns a Page.
// It returns a nil page if there is a network error.
// It will also return an error if the response is not 2xx,
// but the returned page is non-nil, and you can parse the error body.
func (scraper *Scraper) POST(url string, form url.Values) (*Page, error) {
	scraper.history.Add(scraper.b + url)
	resp, err := scraper.c.PostForm(scraper.b+url, form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("scraper: non 2xx response code: %d: %s", resp.StatusCode, resp.Status)
	}
	return &Page{resp: resp}, nil
}

// Debug enables debugging messages to standard error stream.
func (scraper *Scraper) Debug(enabled bool) *Scraper {
	scraper.debug = enabled
	return scraper
}

// SetUA allows one to change the default user agent used by the Scraper.
func (scraper *Scraper) SetUA(userAgent string) *Scraper {
	scraper.c.Transport.(*transport).ua = userAgent
	return scraper
}

// BaseURL can be used to setup Scraper base URL,
// that will then be a prefix used by Get and Post methods.
func (scraper *Scraper) BaseURL(baseURL string) *Scraper {
	scraper.b = baseURL
	return scraper
}

func (scraper *Scraper) History() *History {
	return scraper.history
}

func (scraper *Scraper) checkRedirect(req *http.Request, via []*http.Request) error {
	log.Printf("Redirecting to: %v (via %v)", req, via)
	scraper.history.Add(req.URL.String())
	if len(via) > 10 {
		return ErrTooManyRedirects
	}
	return nil
}
