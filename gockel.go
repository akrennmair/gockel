package main

import (
	"fmt"
	"os"
	"json"
	"io/ioutil"
	oauth "github.com/akrennmair/goauth"
)

func main() {

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

	model := NewModel(tapi)
	updatechan := model.GetUpdateChannel()
	newtweetchan := model.GetNewTweetChannel()
	go model.Run()

	ctrl := NewController(newtweetchan)
	viewchan := ctrl.GetViewChannel()
	go ctrl.Run()

	ui := NewUserInterface(viewchan, updatechan)
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
