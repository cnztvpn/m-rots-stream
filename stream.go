package stream

import (
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
		fetch:   NewFetch(c.Auth),
	}
}
