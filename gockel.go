package main

import (
	oauth "github.com/hokapoka/goauth"
	"fmt"
	"io/ioutil"
	"json"
)

type Timeline struct {
	Tweets []Tweet
}

type Tweet struct {
	Favorited bool
	In_reply_to_status_id *int64
	//Retweet_count *string
	In_reply_to_screen_name *string
	Place *PlaceDesc
	Truncated bool
	User *TwitterUser
	Retweeted bool
	In_reply_to_status_id_str *string
	In_reply_to_user_id_str *string
	In_reply_to_user_id *int64
	Source *string
	Id *int64
	Id_str *string
	//Coordinates *TODO
	Text *string
	Created_at *string
}

type TwitterUser struct {
	Protected bool
	Listed_count int
	Name *string
	Verified bool
	Lang *string
	Time_zone *string
	Description *string
	Location *string
	Statuses_count int
	Url *string
	Screen_name *string
	Follow_request_sent bool
	Following bool
	Friends_count *int64
	Favourites_count *int64
	Followers_count *int64
	Id *int64
	Id_str *string
}

type PlaceDesc struct {
	Name *string
	Full_name *string
	Url *string
	Country_code *string
}

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

	foo, geterr := goauthcon.Get("https://api.twitter.com/1/statuses/home_timeline.json", oauth.Params{}, at )

	if geterr != nil {
		fmt.Println(geterr.String())
	}

	body, _ := ioutil.ReadAll(foo.Body)

	var home_tl Timeline

	if jsonerr := json.Unmarshal(body, &home_tl.Tweets); jsonerr == nil {
		for _, tweet := range home_tl.Tweets {
			fmt.Printf("[%s] %s\n", *tweet.User.Screen_name, *tweet.Text)
		}
	} else {
		fmt.Println(jsonerr.String())
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
