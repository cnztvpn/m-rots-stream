package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/logrusorgru/aurora/v3"
	lowe "github.com/m-rots/bernard"
	ds "github.com/m-rots/bernard/datastore"
	"github.com/m-rots/stream"
	"github.com/m-rots/stubbs"
	"gopkg.in/yaml.v2"
)

type config struct {
	AuthPath     string `yaml:"auth"`
	DatabasePath string `yaml:"database"`
	Port         int    `yaml:"port"`
	DriveID      string `yaml:"drive"`
	Depth        int    `yaml:"depth"`
	FilmsID      string `yaml:"films"`
	ShowsID      string `yaml:"shows"`
}

func main() {
	file, err := os.Open("./config.yml")
	ifErrorThenExit(err,
		"could not open `config.yml`",
		[]string{
			"you can create a `config.yml` file in the current directory",
		},
	)

	c := config{}
	decoder := yaml.NewDecoder(file)
	decoder.SetStrict(true)
	err = decoder.Decode(&c)
	ifErrorThenExit(err,
		"invalid config file / missing values",
		[]string{
			"make sure your `config.yml` file contains valid YAML syntax",
			"and provides all required fields",
		},
	)

	auth := newAuth(c.AuthPath, []string{"https://www.googleapis.com/auth/drive.readonly"})

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
	bernard := lowe.New(auth, store, lowe.WithSafeSleep(0*time.Minute))

	_, err = store.PageToken(c.DriveID)
	if err != nil {
		if !errors.Is(err, ds.ErrFullSync) {
			ifErrorThenExit(err,
				"unexpected error while fetching pageToken from Bernard",
				[]string{
					"please open an issue in the github repo",
					"something went terribly wrong :(",
				},
			)
		}

		// print info on 2 minute safe sync
		fmt.Printf("%s not synchronised yet, starting synchronisation...\n", c.DriveID)
		err = bernard.FullSync(c.DriveID)
		handleSyncError(c, err)
	} else {
		fmt.Println("Performing partial sync...")
		err = bernard.PartialSync(c.DriveID)
		handleSyncError(c, err)
	}

	fmt.Println("Finished synchronisation!")
	fmt.Printf("Stream listening on port %d\n", c.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", c.Port), s.Handler())
}

type serviceAccount struct {
	Email      string `json:"client_email"`
	PrivateKey string `json:"private_key"`
}

func newAuth(path string, scopes []string) *stubbs.Stubbs {
	file, err := os.Open(path)
	ifErrorThenExit(err,
		fmt.Sprintf("could not open `%s`", path),
		[]string{
			"make sure the `auth` field in your config file",
			"points to an existing JSON service account key file",
		},
	)

	decoder := json.NewDecoder(file)
	sa := new(serviceAccount)

	err = decoder.Decode(sa)
	ifErrorThenExit(err,
		fmt.Sprintf("invalid JSON syntax in `%s`", path),
		[]string{
			"make sure the `auth` field in your config file",
			"points to an existing JSON service account key file",
		},
	)

	priv, err := stubbs.ParseKey(sa.PrivateKey)
	ifErrorThenExit(err,
		fmt.Sprintf("invalid private key in `%s`", path),
		[]string{
			"make sure the `auth` field in your config file",
			"points to an existing JSON service account key file",
		},
	)

	return stubbs.New(sa.Email, &priv, scopes, 3600)
}

func ifErrorThenExit(err error, msg string, help []string) {
	if err != nil {
		fmt.Printf("%s%s\n", aurora.BrightRed("error"), aurora.Bold(": "+msg))
		fmt.Printf("  ---> %s\n", err.Error())

		fmt.Printf("\n  %s\n", aurora.Bold("help:"))
		for _, h := range help {
			fmt.Printf("  %s\n", h)
		}
		os.Exit(1)
	}
}

func handleSyncError(c config, err error) {
	if errors.Is(err, lowe.ErrNotFound) {
		ifErrorThenExit(err,
			fmt.Sprintf("cannot access shared drive `%s`", c.DriveID),
			[]string{
				fmt.Sprintf("make sure your service account has read access to `%s`", c.DriveID),
			},
		)
	}

	if errors.Is(err, lowe.ErrInvalidCredentials) {
		ifErrorThenExit(err,
			fmt.Sprintf("service account is invalid `%s`", c.AuthPath),
			[]string{
				fmt.Sprintf("maybe you have edited your service account file `%s`", c.AuthPath),
				"if that is the case, please restore the key to its original state",
				"",
				"additionally, you may have deleted the account (or key) from the cloud console",
				"if that is the case, please create a new account / key",
			},
		)
	}

	ifErrorThenExit(err,
		"unexpected error while performing full-sync",
		[]string{
			"¯\\_(ツ)_/¯",
		},
	)
}
