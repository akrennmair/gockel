package main

import (
	"fmt"
	"os"
	"json"
	"io/ioutil"
	"time"
	oauth "github.com/hokapoka/goauth"
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

	last_id := int64(0)

	for {

		home_tl, err := tapi.HomeTimeline(0, last_id)

		if err != nil {
			fmt.Println(err.String())
		} else {

			for _, tweet := range home_tl.Tweets {
				fmt.Printf("[id=%v] [%s] %s\n", *tweet.Id, *tweet.User.Screen_name, *tweet.Text)
			}

			if len(home_tl.Tweets) > 0 && home_tl.Tweets[0].Id != nil {
				last_id = *home_tl.Tweets[0].Id
				fmt.Printf("last_id = %v\n", last_id)
			}
		}

		time.Sleep(20e9)
	}

//	foo, posterr := goauthcon.Post(
//		"http://api.twitter.com/1/statuses/update.json",
//		oauth.Params{
//			&oauth.Pair{
//				Key:"status",
//				Value:"Test posting using Gockel prototype",
//			},
//		}, at )
//
//	fmt.Printf("foo = %v\n", foo)
//
//	if posterr != nil {
//		fmt.Println(err.String())
//		return
//	}
//
//	fmt.Println("Twitter Status is updated")
}

func SaveAccessToken(at *oauth.AccessToken) os.Error {
	data, marshalerr := json.Marshal(at)
	if marshalerr != nil {
		return marshalerr
	}

	f, ferr := os.OpenFile("access_token.json", os.O_WRONLY | os.O_CREATE, 0600)
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
