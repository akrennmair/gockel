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

type UserList struct {
	Users []TwitterUser
}

type UserIdList struct {
	Ids []int64
}

type Tweet struct {
	Favorited *bool
	In_reply_to_status_id *int64
	Retweet_count interface{}
	In_reply_to_screen_name *string
	Place *PlaceDesc
	Truncated *bool
	User *TwitterUser
	Retweeted *bool
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
	Protected *bool
	Listed_count int
	Name *string
	Verified *bool
	Lang *string
	Time_zone *string
	Description *string
	Location *string
	Statuses_count int
	Url *string
	Screen_name *string
	Follow_request_sent *bool
	Following *bool
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
	return tapi.get_timeline("home_timeline", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) Mentions(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("mentions", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) PublicTimeline(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("public_timeline", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) RetweetedByMe(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_by_me", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) RetweetedToMe(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_to_me", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) RetweetsOfMe(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweets_of_me", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) UserTimeline(screen_name string, count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("user_timeline", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}(),
		func() *oauth.Pair {
			if screen_name != "" {
				return &oauth.Pair{ "screen_name", screen_name }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) RetweetedToUser(screen_name string, count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_to_user", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}(),
		func() *oauth.Pair {
			if screen_name != "" {
				return &oauth.Pair{ "screen_name", screen_name }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) RetweetedByUser(screen_name string, count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_by_user", 
		func() *oauth.Pair { 
			if count != 0 { 
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}(),
		func() *oauth.Pair { 
			if since_id != 0 { 
				return &oauth.Pair{ "since_id", strconv.Itoa64(since_id) }
			}
			return nil
		}(),
		func() *oauth.Pair {
			if screen_name != "" {
				return &oauth.Pair{ "screen_name", screen_name }
			}
			return nil
		}() )
}

func(tapi *TwitterAPI) RetweetedBy(tweet_id int64, count uint) (*UserList, os.Error) {
	jsondata, err := tapi.get_statuses(strconv.Itoa64(tweet_id) + "/retweeted_by",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}() )
	if err != nil {
		return nil, err
	}

	ul := &UserList{}

	if jsonerr := json.Unmarshal(jsondata, &ul.Users); jsonerr != nil {
		return nil, jsonerr
	}

	return ul, nil
}

func(tapi *TwitterAPI) RetweetedByIds(tweet_id int64, count uint) (*UserIdList, os.Error) {
	jsondata, err := tapi.get_statuses(strconv.Itoa64(tweet_id) + "/retweeted_by/ids",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{ "count", strconv.Uitoa(count) }
			}
			return nil
		}() )
	if err != nil {
		return nil, err
	}

	uidl := &UserIdList{}

	if jsonerr := json.Unmarshal(jsondata, &uidl.Ids); jsonerr != nil {
		return nil, jsonerr
	}

	return uidl, nil
}

func(tapi *TwitterAPI) get_timeline(tl_name string, p ...*oauth.Pair) (*Timeline, os.Error) {
	jsondata, err := tapi.get_statuses(tl_name, p...)
	if err != nil {
		return nil, err
	}

	tl := &Timeline{}

	if jsonerr := json.Unmarshal(jsondata, &tl.Tweets); jsonerr != nil {
		return nil, jsonerr
	}

	return tl, nil
}

func(tapi *TwitterAPI) get_statuses(id string, p ...*oauth.Pair) ([]byte, os.Error) {
	var params oauth.Params
	for _, x := range p {
		if x != nil {
			params.Add(x)
		}
	}

	resp, geterr := tapi.authcon.Get("https://api.twitter.com/1/statuses/" + id + ".json", params, tapi.access_token)
	if geterr != nil {
		return nil, geterr
	}

	return ioutil.ReadAll(resp.Body)
}
