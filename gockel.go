package main

import (
	oauth "github.com/hokapoka/goauth"
	"fmt"
)

var goauthcon *oauth.OAuthConsumer

func main() {
	goauthcon = &oauth.OAuthConsumer{
		Service:"twitter",
		RequestTokenURL:"http://twitter.com/oauth/request_token",
		AccessTokenURL:"http://twitter.com/oauth/access_token",
		AuthorizationURL:"http://twitter.com/oauth/authorize",
		ConsumerKey:"sDggzGbHbyAfl5fJ87XOCA",
		ConsumerSecret:"MOCQDL7ot7qIxMwYL5x1mMAqiYBYxNTxPWS6tc6hw",
		CallBackURL:"oob",
	}

	s, rt, err := goauthcon.GetRequestAuthorizationURL()
	if err != nil {
		fmt.Println(err.String())
		return
	}

	var pin string
	fmt.Printf("Open %s\n", s)
	fmt.Printf("PIN Number: ")
	fmt.Scanln(&pin)

	at := goauthcon.GetAccessToken(rt.Token, pin)

	fmt.Printf("at = %v\n", at)

	foo, posterr := goauthcon.Post(
		"http://api.twitter.com/1/statuses/update.json",
		oauth.Params{
			&oauth.Pair{
				Key:"status",
				Value:"Test posting using Gockel prototype",
			},
		}, at )

	fmt.Printf("foo = %v\n", foo)

	if posterr != nil {
		fmt.Println(err.String())
		return
	}

	fmt.Println("Twitter Status is updated")
}
