package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	lowe "github.com/m-rots/bernard"
	ds "github.com/m-rots/bernard/datastore"
	"github.com/m-rots/stream"
	"github.com/m-rots/stubbs"
	"gopkg.in/yaml.v2"
)

type config struct {
	AuthPath     string `yaml:"auth"`
	DatabasePath string `yaml:"database"`
	DriveID      string `yaml:"drive"`
	Depth        int    `yaml:"depth"`
	FilmsID      string `yaml:"films"`
	ShowsID      string `yaml:"shows"`
}

func main() {
	file, err := os.Open("./config.yml")
	if err != nil {
		panic(err)
	}

	c := new(config)
	decoder := yaml.NewDecoder(file)
	decoder.SetStrict(true)
	err = decoder.Decode(c)
	if err != nil {
		panic(err)
	}

	auth, err := newAuth(c.AuthPath, []string{"https://www.googleapis.com/auth/drive.readonly"})
	if err != nil {
		panic(err)
	}

	store, err := stream.NewStore(c.DatabasePath)
	if err != nil {
		panic(err)
	}

	streamConf := stream.Config{
		Depth:   c.Depth,
		FilmsID: c.FilmsID,
		ShowsID: c.ShowsID,

		Auth:  auth,
		Store: store,
	}

	s := stream.NewStream(streamConf)
	if err != nil {
		panic(err)
	}

	bernard := lowe.New(auth, store,
		lowe.WithSafeSleep(0*time.Minute))

	_, err = store.PageToken(c.DriveID)
	if err != nil {
		if !errors.Is(err, ds.ErrFullSync) {
			panic(err)
		}

		// print info on 2 minute safe sync
		fmt.Printf("%s not synchronised yet, starting synchronisation...\n", c.DriveID)
		err = bernard.FullSync(c.DriveID)
		if err != nil {
			panic(err)
		}

		fmt.Println("Finished synchronisation!")
	} else {
		fmt.Println("Performing partial sync...")
		err = bernard.PartialSync(c.DriveID)
		if err != nil {
			panic(err)
		}
		fmt.Println("Finished synchronisation!")
	}

	fmt.Println("Stream is listening on port 3000!")
	http.ListenAndServe(":3000", s.Handler())
}

type serviceAccount struct {
	Email      string `json:"client_email"`
	PrivateKey string `json:"private_key"`
}

func newAuth(path string, scopes []string) (*stubbs.Stubbs, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("stream: cannot open service account file")
	}

	decoder := json.NewDecoder(file)
	sa := new(serviceAccount)

	if decoder.Decode(sa) != nil {
		return nil, errors.New("stream: cannot decode the service account's JSON file")
	}

	priv, err := stubbs.ParseKey(sa.PrivateKey)
	if err != nil {
		return nil, errors.New("stream: invalid private key")
	}

	return stubbs.New(sa.Email, &priv, scopes, 3600), nil
}
