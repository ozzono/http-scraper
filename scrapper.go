// Copyright 2015 Ronoaldo JLP <ronoaldo@gmail.com>
// Licensed under the Apache License, Version 2.0

package scrapper

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

var (
	// ErrTooManyRedirects is returned when the scrapper reaches more than 10 redirects.
	ErrTooManyRedirects = errors.New("scrapper: too many redirects")
)

// Scrapper implements a statefull HTTP client for interacting with websites.
type Scrapper struct {
	b     string
	j     *cookiejar.Jar
	c     *http.Client
	debug bool

	// lastURL records the last seen URL using the CheckRedirect function.
	// TODO(ronoaldo): change to a history of recent URLs.
	history *History
}

// New initializes a new Scrapper with an in-memory cookie management.
func New() *Scrapper {
	return ReuseClient(http.DefaultClient)
}

// CustomNew initializes a new Bot with an in-memory cookie management and with custom http.Client.
func CustomNew(c *http.Client) *Scrapper {
	return ReuseClient(c)
}

func ReuseClient(c *http.Client) *Scrapper {
	jar, err := cookiejar.New(nil)
	if err != nil {
		// Currently, cookiejar.Nil never returns an error
		panic(err)
	}
	c.Jar = jar
	scrapper := &Scrapper{
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
		b: scrapper,
	}
	scrapper.c.Transport = t
	scrapper.c.CheckRedirect = scrapper.checkRedirect
	return scrapper
}

// Do sends the HTTP request using the http.Client.Do.
// It returns a nil page if there is a network error.
// It will also return an error if the response is not 2xx,
// but the returned page is non-nil, and you can parse the error body.
func (scrapper *Scrapper) Do(req *http.Request) (*Page, error) {
	scrapper.history.Add(scrapper.b + req.URL.String())
	resp, err := scrapper.c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("scrapper: non 2xx response code: %d: %s", resp.StatusCode, resp.Status)
	}
	return &Page{resp: resp}, nil
}

// GET performs the HTTP GET to the provided URL and returns a Page.
// It returns a nil page if there is a network error.
// It will also return an error if the response is not 2xx,
// but the returned page is non-nil, and you can parse the error body.
func (scrapper *Scrapper) GET(url string) (*Page, error) {
	scrapper.history.Add(scrapper.b + url)
	resp, err := scrapper.c.Get(scrapper.b + url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("scrapper: non 2xx response code: %d: %s", resp.StatusCode, resp.Status)
	}
	return &Page{resp: resp}, nil
}

// POST performs an HTTP POST to the provided URL,
// using the form as a payload, and returns a Page.
// It returns a nil page if there is a network error.
// It will also return an error if the response is not 2xx,
// but the returned page is non-nil, and you can parse the error body.
func (scrapper *Scrapper) POST(url string, form url.Values) (*Page, error) {
	scrapper.history.Add(scrapper.b + url)
	resp, err := scrapper.c.PostForm(scrapper.b+url, form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("scrapper: non 2xx response code: %d: %s", resp.StatusCode, resp.Status)
	}
	return &Page{resp: resp}, nil
}

// Debug enables debugging messages to standard error stream.
func (scrapper *Scrapper) Debug(enabled bool) *Scrapper {
	scrapper.debug = enabled
	return scrapper
}

// SetUA allows one to change the default user agent used by the Scrapper.
func (scrapper *Scrapper) SetUA(userAgent string) *Scrapper {
	scrapper.c.Transport.(*transport).ua = userAgent
	return scrapper
}

// BaseURL can be used to setup Scrapper base URL,
// that will then be a prefix used by Get and Post methods.
func (scrapper *Scrapper) BaseURL(baseURL string) *Scrapper {
	scrapper.b = baseURL
	return scrapper
}

func (scrapper *Scrapper) History() *History {
	return scrapper.history
}

func (scrapper *Scrapper) checkRedirect(req *http.Request, via []*http.Request) error {
	log.Printf("Redirecting to: %v (via %v)", req, via)
	scrapper.history.Add(req.URL.String())
	if len(via) > 10 {
		return ErrTooManyRedirects
	}
	return nil
}
