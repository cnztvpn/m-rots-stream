package stream

import (
	"crypto/tls"
	"net/http"
	"time"

	lowe "github.com/m-rots/bernard"
)

type Config struct {
	Depth   int
	FilmsID string
	ShowsID string

	Auth  lowe.Authenticator
	Store Store
}

type Stream struct {
	depth   int
	filmsID string
	showsID string

	fetch fetch
	store Store
}

func NewStream(c Config) Stream {
	if c.Depth < 1 {
		c.Depth = 1
	}

	return Stream{
		depth:   c.Depth,
		filmsID: c.FilmsID,
		showsID: c.ShowsID,
		store:   c.Store,
		fetch:   newFetch(c.Auth),
	}
}

func newFetch(auth lowe.Authenticator) fetch {
	const baseURL string = "https://www.googleapis.com/drive/v3"

	return fetch{
		auth:    auth,
		baseURL: baseURL,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				// An empty map for TLSNextProto disables HTTP/2.
				// Unfortunately, memory use and stability get hammered when using HTTP/2.
				TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			},
			Timeout: time.Minute * 5,
		},
	}
}
