package main

import (
	"fmt"
	"os"
	"json"
	"io/ioutil"
	"flag"
	"log"
	oauth "github.com/akrennmair/goauth"
)

const (
	PROGRAM_NAME = "gockel"
	PROGRAM_VERSION = "0.0"
	PROGRAM_URL = "https://github.com/akrennmair/gockel"
)

type DevNullWriter int

func (w *DevNullWriter) Write(b []byte) (n int, err os.Error) {
	return len(b), nil
}

func main() {
	var logfile *string = flag.String("log", "", "logfile")

	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	devnull := new(DevNullWriter)
	log.SetOutput(devnull)

	if *logfile != "" {
		if f, err := os.OpenFile(*logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600); err == nil {
			defer f.Close()
			log.SetOutput(f)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: couldn't open logfile for writing: %v\n", err)
		}
	}


	tapi := NewTwitterAPI("sDggzGbHbyAfl5fJ87XOCA", "MOCQDL7ot7qIxMwYL5x1mMAqiYBYxNTxPWS6tc6hw")

	at, aterr := LoadAccessToken()

	if aterr == nil {
		tapi.SetAccessToken(at)
	} else {
		auth_url, err := tapi.GetRequestAuthorizationURL()
		if err != nil {
			fmt.Println(err.String())
			return
		}

		var pin string
		fmt.Printf("Open %s\n", auth_url)
		fmt.Printf("PIN Number: ")
		fmt.Scanln(&pin)

		tapi.SetPIN(pin)

		if saveerr := SaveAccessToken(tapi.GetAccessToken()); saveerr != nil {
			fmt.Printf("saving access token failed: %s\n", saveerr.String())
			return
		}
	}

	cmdchan := make(chan TwitterCommand, 1)
	newtweetchan := make(chan []*Tweet, 1)
	lookupchan := make(chan TweetRequest, 1)
	uiactionchan := make(chan UserInterfaceAction, 10)

	model := NewModel(tapi, cmdchan, newtweetchan, lookupchan, uiactionchan)
	go model.Run()

	ui := NewUserInterface(cmdchan, newtweetchan, lookupchan, uiactionchan)
	go ui.Run()

	ui.InputLoop()
}

func SaveAccessToken(at *oauth.AccessToken) os.Error {
	data, marshalerr := json.Marshal(at)
	if marshalerr != nil {
		return marshalerr
	}

	f, ferr := os.OpenFile("access_token.json", os.O_WRONLY|os.O_CREATE, 0600)
	if ferr != nil {
		return ferr
	}
	defer f.Close()

	_, werr := f.Write(data)
	if werr != nil {
		return werr
	}

	return nil
}

func LoadAccessToken() (*oauth.AccessToken, os.Error) {
	f, ferr := os.Open("access_token.json")
	if ferr != nil {
		return nil, ferr
	}
	defer f.Close()

	data, readerr := ioutil.ReadAll(f)
	if readerr != nil {
		return nil, readerr
	}

	at := &oauth.AccessToken{}

	err := json.Unmarshal(data, at)
	if err != nil {
		return nil, err
	}

	return at, nil
}
