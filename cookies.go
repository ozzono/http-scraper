package scrapper

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type CookieJar struct {
	Data map[string][]*http.Cookie
}

func (scrapper *Scrapper) EncodeCookies() ([]byte, error) {
	history := scrapper.History().Entries()
	jar := &CookieJar{
		Data: make(map[string][]*http.Cookie),
	}
	for i := range history {
		u, err := url.Parse(history[i])
		if err != nil {
			log.Printf("Invalid URL in history! Skipping cookies from it: " + history[i])
			continue
		}
		jar.Data[u.String()] = scrapper.j.Cookies(u)
	}
	return json.MarshalIndent(jar, "", "  ")
}

func (scrapper *Scrapper) DecodeCookies(cookies []byte) error {
	jar := &CookieJar{
		Data: make(map[string][]*http.Cookie),
	}
	if err := json.Unmarshal(cookies, jar); err != nil {
		return err
	}

	for k := range jar.Data {
		v := jar.Data[k]
		u, err := url.Parse(k)
		if err != nil {
			log.Printf("Invalid URL from cookies! Skipping: " + k)
			continue
		}
		scrapper.j.SetCookies(u, v)
		scrapper.History().Add(u.String())
	}
	return nil
}

func (scrapper *Scrapper) SetCookie(c *http.Cookie) {
	var host = c.Domain
	if strings.HasPrefix(host, ".") {
		host = host[1:]
	}
	u := &url.URL{
		Scheme: "http",
		Host:   host,
	}
	log.Printf("Setting cookie: u=%v ; value=%v", u, c)
	scrapper.j.SetCookies(u, []*http.Cookie{c})
	scrapper.History().Add(u.String())
}
