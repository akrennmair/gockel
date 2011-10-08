package main

import (
	oauth "github.com/hokapoka/goauth"
	"json"
	"os"
	"io/ioutil"
	"strconv"
)

type Timeline struct {
	Tweets []Tweet
}

type Tweet struct {
	Favorited bool
	In_reply_to_status_id *int64
	Retweet_count interface{}
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

const request_token_url = "http://twitter.com/oauth/request_token"
const access_token_url = "http://twitter.com/oauth/access_token"
const authorization_url = "http://twitter.com/oauth/authorize"

type TwitterAPI struct {
	authcon *oauth.OAuthConsumer
	access_token *oauth.AccessToken
	request_token *oauth.RequestToken
}

func NewTwitterAPI(consumer_key, consumer_secret string) *TwitterAPI {
	tapi := &TwitterAPI { 
				authcon: &oauth.OAuthConsumer {
					Service:"twitter",
					RequestTokenURL: request_token_url,
					AccessTokenURL: access_token_url,
					AuthorizationURL: authorization_url,
					ConsumerKey: consumer_key,
					ConsumerSecret: consumer_secret,
					CallBackURL: "oob",
				},
			}
	return tapi
}

func(tapi *TwitterAPI) GetRequestAuthorizationURL() (string, os.Error) {
	s, rt, err := tapi.authcon.GetRequestAuthorizationURL()
	tapi.request_token = rt
	return s, err
}

func(tapi *TwitterAPI) SetPIN(pin string) {
	tapi.access_token = tapi.authcon.GetAccessToken(tapi.request_token.Token, pin)
}

func(tapi *TwitterAPI) SetAccessToken(at *oauth.AccessToken) {
	tapi.access_token = at
}

func(tapi *TwitterAPI) GetAccessToken() *oauth.AccessToken {
	return tapi.access_token
}

func(tapi *TwitterAPI) HomeTimeline(count uint, since_id int64) (*Timeline, os.Error) {
	var params oauth.Params
	if count != 0 {
		params.Add(&oauth.Pair{ "count", strconv.Uitoa(count) })
	}
	if since_id != 0 {
		params.Add(&oauth.Pair{ "since_id", strconv.Itoa64(since_id) })
	}

	resp, geterr := tapi.authcon.Get("https://api.twitter.com/1/statuses/home_timeline.json", params, tapi.access_token)
	if geterr != nil {
		return nil, geterr
	}

	bodydata, readerr := ioutil.ReadAll(resp.Body)
	if readerr != nil {
		return nil, readerr
	}

	home_tl := &Timeline{}

	if jsonerr := json.Unmarshal(bodydata, &home_tl.Tweets); jsonerr != nil {
		return nil, jsonerr
	}

	return home_tl, nil
}
