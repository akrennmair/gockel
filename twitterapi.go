package main

import (
	oauth "github.com/akrennmair/goauth"
	goconf "goconf.googlecode.com/hg"
	"json"
	"os"
	"io/ioutil"
	"strconv"
	"fmt"
	"http"
	"strings"
	"time"
	"bufio"
	"bytes"
	"log"
)

type Timeline struct {
	Tweets []*Tweet
}

type UserList struct {
	Users []TwitterUser
}

type UserIdList struct {
	Ids []int64
}

type Tweet struct {
	Favorited                 *bool
	In_reply_to_status_id     *int64
	Retweet_count             interface{}
	In_reply_to_screen_name   *string
	Place                     *PlaceDesc
	Truncated                 *bool
	User                      *TwitterUser
	Retweeted                 *bool
	In_reply_to_status_id_str *string
	In_reply_to_user_id_str   *string
	In_reply_to_user_id       *int64
	Source                    *string
	Id                        *int64
	Id_str                    *string
	//Coordinates *TODO
	Text       *string
	Created_at *string
}

type TwitterUser struct {
	Protected           *bool
	Listed_count        int
	Name                *string
	Verified            *bool
	Lang                *string
	Time_zone           *string
	Description         *string
	Location            *string
	Statuses_count      int
	Url                 *string
	Screen_name         *string
	Follow_request_sent *bool
	Following           *bool
	Friends_count       *int64
	Favourites_count    *int64
	Followers_count     *int64
	Id                  *int64
	Id_str              *string
}

type PlaceDesc struct {
	Name         *string
	Full_name    *string
	Url          *string
	Country_code *string
}

type TwitterEvent struct {
	Delete *WhatEvent
}

type WhatEvent struct {
	Status *EventDetail
}

type EventDetail struct {
	Id          *int64
	Id_str      *string
	User_id     *int64
	User_id_str *string
}

type Configuration struct {
	Characters_reserved_per_media *int64
	Max_media_per_upload          *int64
	Short_url_length_https        *int64
	Short_url_length              *int64
}

const (
	request_token_url = "https://twitter.com/oauth/request_token"
	access_token_url  = "https://twitter.com/oauth/access_token"
	authorization_url = "https://twitter.com/oauth/authorize"

	INITIAL_NETWORK_WAIT int64 = 250e6 // 250 milliseconds
	INITIAL_HTTP_WAIT    int64 = 10e9  // 10 seconds
	MAX_NETWORK_WAIT     int64 = 16e9  // 16 seconds
	MAX_HTTP_WAIT        int64 = 240e9 // 240 seconds
)

type TwitterAPI struct {
	authcon         *oauth.OAuthConsumer
	config          *goconf.ConfigFile
	access_token    *oauth.AccessToken
	request_token   *oauth.RequestToken
	ratelimit_rem   uint
	ratelimit_limit uint
	ratelimit_reset int64
}

func NewTwitterAPI(consumer_key, consumer_secret string, cfg *goconf.ConfigFile) *TwitterAPI {
	tapi := &TwitterAPI{
		authcon: &oauth.OAuthConsumer{
			Service:          "twitter",
			RequestTokenURL:  request_token_url,
			AccessTokenURL:   access_token_url,
			AuthorizationURL: authorization_url,
			ConsumerKey:      consumer_key,
			ConsumerSecret:   consumer_secret,
			UserAgent:        PROGRAM_NAME + "/" + PROGRAM_VERSION + " (" + PROGRAM_URL + ")",
			Timeout:          60e9, // 60 second default timeout
			CallBackURL:      "oob",
		},
		config: cfg,
	}
	if tapi.config != nil {
		if timeout, err := tapi.config.GetInt("default", "http_timeout"); err == nil && timeout > 0 {
			tapi.authcon.Timeout = int64(timeout) * 1e9
		}
	}
	return tapi
}

func (tapi *TwitterAPI) GetRequestAuthorizationURL() (string, os.Error) {
	s, rt, err := tapi.authcon.GetRequestAuthorizationURL()
	tapi.request_token = rt
	return s, err
}

func (tapi *TwitterAPI) GetRateLimit() (remaining uint, limit uint, reset int64) {
	curtime, _, _ := os.Time()
	return tapi.ratelimit_rem, tapi.ratelimit_limit, tapi.ratelimit_reset - curtime
}

func (tapi *TwitterAPI) SetPIN(pin string) {
	tapi.access_token = tapi.authcon.GetAccessToken(tapi.request_token.Token, pin)
}

func (tapi *TwitterAPI) SetAccessToken(at *oauth.AccessToken) {
	tapi.access_token = at
}

func (tapi *TwitterAPI) GetAccessToken() *oauth.AccessToken {
	return tapi.access_token
}

func (tapi *TwitterAPI) HomeTimeline(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("home_timeline",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) Mentions(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("mentions",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) PublicTimeline(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("public_timeline",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) RetweetedByMe(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_by_me",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) RetweetedToMe(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_to_me",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) RetweetsOfMe(count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweets_of_me",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) UserTimeline(screen_name string, count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("user_timeline",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if screen_name != "" {
				return &oauth.Pair{"screen_name", screen_name}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) RetweetedToUser(screen_name string, count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_to_user",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if screen_name != "" {
				return &oauth.Pair{"screen_name", screen_name}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) RetweetedByUser(screen_name string, count uint, since_id int64) (*Timeline, os.Error) {
	return tapi.get_timeline("retweeted_by_user",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if since_id != 0 {
				return &oauth.Pair{"since_id", strconv.Itoa64(since_id)}
			}
			return nil
		}(),
		func() *oauth.Pair {
			if screen_name != "" {
				return &oauth.Pair{"screen_name", screen_name}
			}
			return nil
		}())
}

func (tapi *TwitterAPI) RetweetedBy(tweet_id int64, count uint) (*UserList, os.Error) {
	jsondata, err := tapi.get_statuses(strconv.Itoa64(tweet_id)+"/retweeted_by",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}())
	if err != nil {
		return nil, err
	}

	ul := &UserList{}

	if jsonerr := json.Unmarshal(jsondata, &ul.Users); jsonerr != nil {
		return nil, jsonerr
	}

	return ul, nil
}

func (tapi *TwitterAPI) RetweetedByIds(tweet_id int64, count uint) (*UserIdList, os.Error) {
	jsondata, err := tapi.get_statuses(strconv.Itoa64(tweet_id)+"/retweeted_by/ids",
		func() *oauth.Pair {
			if count != 0 {
				return &oauth.Pair{"count", strconv.Uitoa(count)}
			}
			return nil
		}())
	if err != nil {
		return nil, err
	}

	uidl := &UserIdList{}

	if jsonerr := json.Unmarshal(jsondata, &uidl.Ids); jsonerr != nil {
		return nil, jsonerr
	}

	return uidl, nil
}

func (tapi *TwitterAPI) Update(tweet Tweet) (*Tweet, os.Error) {
	params := oauth.Params{
		&oauth.Pair{
			Key:   "status",
			Value: *tweet.Text,
		},
	}
	if tweet.In_reply_to_status_id != nil && *tweet.In_reply_to_status_id != int64(0) {
		params = append(params, &oauth.Pair{"in_reply_to_status_id", strconv.Itoa64(*tweet.In_reply_to_status_id)})
	}
	resp, err := tapi.authcon.Post("https://api.twitter.com/1/statuses/update.json", params, tapi.access_token)
	if err != nil {
		return nil, err
	}

	tapi.UpdateRatelimit(resp.Header)

	if resp.StatusCode == 403 {
		return nil, os.NewError(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	newtweet := &Tweet{}
	if jsonerr := json.Unmarshal(data, newtweet); jsonerr != nil {
		return nil, jsonerr
	}

	return newtweet, nil
}

func (tapi *TwitterAPI) Retweet(tweet Tweet) (*Tweet, os.Error) {
	resp, err := tapi.authcon.Post(fmt.Sprintf("https://api.twitter.com/1/statuses/retweet/%d.json", *tweet.Id), oauth.Params{}, tapi.access_token)
	if err != nil {
		return nil, err
	}

	tapi.UpdateRatelimit(resp.Header)

	if resp.StatusCode == 403 {
		return nil, os.NewError(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	newtweet := &Tweet{}
	if jsonerr := json.Unmarshal(data, newtweet); jsonerr != nil {
		return nil, jsonerr
	}

	return newtweet, nil
}

func (tapi *TwitterAPI) Favorite(tweet Tweet) os.Error {
	resp, err := tapi.authcon.Post(fmt.Sprintf("https://api.twitter.com/1/favorites/create/%d.json", *tweet.Id), oauth.Params{}, tapi.access_token)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return os.NewError(resp.Status)
	}

	return nil
}

func (tapi *TwitterAPI) Follow(screen_name string) os.Error {
	params := oauth.Params{
		&oauth.Pair{
			Key:   "screen_name",
			Value: screen_name,
		},
	}
	resp, err := tapi.authcon.Post("https://api.twitter.com/1/friendships/create.json", params, tapi.access_token)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return os.NewError(resp.Status)
	}

	return nil
}

func (tapi *TwitterAPI) Unfollow(user TwitterUser) os.Error {
	params := oauth.Params{
		&oauth.Pair{
			Key:   "user_id",
			Value: *user.Id_str,
		},
		&oauth.Pair{
			Key:   "screen_name",
			Value: *user.Screen_name,
		},
	}
	resp, err := tapi.authcon.Post("https://api.twitter.com/1/friendships/destroy.json", params, tapi.access_token)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return os.NewError(resp.Status)
	}

	return nil
}

func (tapi *TwitterAPI) DestroyTweet(tweet Tweet) os.Error {
	resp, err := tapi.authcon.Post(fmt.Sprintf("https://api.twitter.com/1/statuses/destroy/%d.json", *tweet.Id), oauth.Params{}, tapi.access_token)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return os.NewError(resp.Status)
	}

	return nil
}

func (tapi *TwitterAPI) Configuration() (*Configuration, os.Error) {
	params := oauth.Params{}
	resp, err := tapi.authcon.Get("https://api.twitter.com/1/help/configuration.json", params, tapi.access_token)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, os.NewError(resp.Status)
	}

	jsondata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	config := &Configuration{}
	if err := json.Unmarshal(jsondata, config); err != nil {
		return nil, err
	}

	return config, nil
}

func (tapi *TwitterAPI) VerifyCredentials() (*TwitterUser, os.Error) {
	params := oauth.Params{
		&oauth.Pair{
			Key:   "skip_status",
			Value: "true",
		},
	}

	resp, err := tapi.authcon.Get("https://api.twitter.com/1/account/verify_credentials.json", params, tapi.access_token)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, os.NewError(resp.Status)
	}

	jsondata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	user := &TwitterUser{}

	if err := json.Unmarshal(jsondata, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (tapi *TwitterAPI) get_timeline(tl_name string, p ...*oauth.Pair) (*Timeline, os.Error) {
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

func (tapi *TwitterAPI) get_statuses(id string, p ...*oauth.Pair) ([]byte, os.Error) {
	var params oauth.Params
	for _, x := range p {
		if x != nil {
			params.Add(x)
		}
	}

	resp, geterr := tapi.authcon.Get("https://api.twitter.com/1/statuses/"+id+".json", params, tapi.access_token)
	if geterr != nil {
		return nil, geterr
	}

	tapi.UpdateRatelimit(resp.Header)

	return ioutil.ReadAll(resp.Body)
}

type HTTPError int

func (e HTTPError) String() string {
	return "HTTP code " + strconv.Itoa(int(e))
}

func (tapi *TwitterAPI) UserStream(tweetchan chan<- []*Tweet, actions chan<- UserInterfaceAction) {
	network_wait := INITIAL_NETWORK_WAIT
	http_wait := INITIAL_HTTP_WAIT
	last_network_backoff := time.Seconds()
	last_http_backoff := time.Seconds()

	for {
		if err := tapi.doUserStream(tweetchan, actions); err != nil {
			log.Printf("user stream returned error: %v", err)
			if _, ok := err.(HTTPError); ok {
				if (time.Seconds() - last_http_backoff) > 1800 {
					http_wait = INITIAL_HTTP_WAIT
				}
				log.Printf("HTTP wait: backing off %d seconds", http_wait/1e9)
				time.Sleep(http_wait)
				if http_wait < MAX_HTTP_WAIT {
					http_wait *= 2
				}
				last_http_backoff = time.Seconds()
			} else {
				if (time.Seconds() - last_network_backoff) > 1800 {
					network_wait = INITIAL_NETWORK_WAIT
				}
				log.Printf("Network wait: backing off %d milliseconds", network_wait/1e6)
				time.Sleep(network_wait)
				if network_wait < MAX_NETWORK_WAIT {
					network_wait += INITIAL_NETWORK_WAIT
				}
				last_network_backoff = time.Seconds()
			}
		}
	}
}

func (tapi *TwitterAPI) doUserStream(tweetchan chan<- []*Tweet, actions chan<- UserInterfaceAction) os.Error {
	resolve_urls := false

	if tapi.config != nil {
		if resolve, err := tapi.config.GetBool("default", "resolve_urls"); err == nil {
			resolve_urls = resolve
		}
	}

	resp, err := tapi.authcon.Get("https://userstream.twitter.com/2/user.json", oauth.Params{}, tapi.access_token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 200 {
		bodydata, _ := ioutil.ReadAll(resp.Body)
		log.Printf("HTTP error: %s", string(bodydata))
		return HTTPError(resp.StatusCode)
	}

	buf := bufio.NewReader(resp.Body)

	for {
		line, err := getLine(buf)
		if err != nil {
			log.Printf("getLine error: %v", err)
			return err
		}
		if len(line) == 0 {
			continue
		}

		if bytes.HasPrefix(line, []byte("{\"delete\":")) {
			action := &TwitterEvent{}

			if err := json.Unmarshal(line, action); err != nil {
				continue
			}

			if action.Delete != nil && action.Delete.Status != nil && action.Delete.Status.Id_str != nil {
				actions <- UserInterfaceAction{DELETE_TWEET, []string{*action.Delete.Status.Id_str}}
			}

		} else {

			newtweet := &Tweet{}
			if err := json.Unmarshal(line, newtweet); err != nil {
				log.Printf("couldn't unmarshal tweet: %v\n", err)
				continue
			}

			// TODO: move this to goroutine if resolving turns out to block everything.
			if resolve_urls {
				newtweet.ResolveURLs()
			}

			if newtweet.Id != nil && newtweet.Text != nil {
				tweetchan <- []*Tweet{newtweet}
			}
		}
	}
	// not reached
	return nil
}

func getLine(buf *bufio.Reader) ([]byte, os.Error) {
	line := []byte{}
	for {
		data, isprefix, err := buf.ReadLine()
		if err != nil {
			return line, err
		}
		line = append(line, data...)
		if !isprefix {
			break
		}
	}
	return line, nil
}

func (tapi *TwitterAPI) UpdateRatelimit(hdrs http.Header) {
	for k, v := range hdrs {
		switch strings.ToLower(k) {
		case "x-ratelimit-limit":
			if limit, err := strconv.Atoui(v[0]); err == nil {
				tapi.ratelimit_limit = limit
			}
		case "x-ratelimit-remaining":
			if rem, err := strconv.Atoui(v[0]); err == nil {
				tapi.ratelimit_rem = rem
			}
		case "x-ratelimit-reset":
			if reset, err := strconv.Atoi64(v[0]); err == nil {
				tapi.ratelimit_reset = reset
			}
		}
	}
}

func (t *Tweet) RelativeCreatedAt() string {
	if t.Created_at == nil {
		return ""
	}

	tt, err := time.Parse(time.RubyDate, *t.Created_at)
	if err != nil {
		return *t.Created_at
	}

	delta := time.LocalTime().Seconds() - tt.Seconds()
	switch {
	case delta < 60:
		return "less than a minute ago"
	case delta < 120:
		return "about a minute ago"
	case delta < 45*60:
		return fmt.Sprintf("about %d minutes ago", delta/60)
	case delta < 120*60:
		return "about an hour ago"
	case delta < 24*60*60:
		return fmt.Sprintf("about %d hours ago", delta/3600)
	case delta < 48*60*60:
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", delta/(3600*24))
}

func longify_url(url string) string {
	if resp, err := http.Head(url); err == nil && resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String()
	}
	return url
}

func (t *Tweet) ResolveURLs() {
	if t.Text != nil {
		*t.Text = FindURLs(*t.Text, longify_url)
	}
}
