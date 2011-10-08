package main

import (
	"fmt"
)


func main() {

	tapi := NewTwitterAPI("sDggzGbHbyAfl5fJ87XOCA", "MOCQDL7ot7qIxMwYL5x1mMAqiYBYxNTxPWS6tc6hw")

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

	home_tl, err := tapi.HomeTimeline()

	if err != nil {
		fmt.Println(err.String())
		return
	}

	for _, tweet := range home_tl.Tweets {
		rt_count, okstr := tweet.Retweet_count.(string)
		if !okstr {
			if rt_count_int, okint := tweet.Retweet_count.(int64); okint {
				rt_count = fmt.Sprintf("%d", rt_count_int)
			}
		}
		fmt.Printf("[%s] %s (%s)\n", *tweet.User.Screen_name, *tweet.Text, rt_count)
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
