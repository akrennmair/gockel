package main

import (
	goconf "github.com/akrennmair/goconf"
	"log"
	"sort"
	"strings"
	"time"
)

type Model struct {
	cmdchan       <-chan interface{}
	newtweetchan  chan<- []*Tweet
	newtweets_int chan []*Tweet
	users         []UserTwitterAPITuple
	cur_user      uint
	tweets        []*Tweet
	tweet_map     map[int64]*Tweet
	lookupchan    <-chan TweetRequest
	uiactionchan  chan<- interface{}
	last_id       int64
	ignored_users []string
}

type TweetRequest struct {
	Status_id int64
	Reply     chan *Tweet
}

type CmdUpdate Tweet

type CmdRetweet Tweet

type CmdFavorite Tweet

type CmdFollow string

type CmdUnfollow TwitterUser

type CmdDestroyTweet Tweet

type CmdSetCurUser uint

func NewModel(users []UserTwitterAPITuple, cc chan interface{}, ntc chan<- []*Tweet, lc <-chan TweetRequest, uac chan<- interface{}, cfg *goconf.ConfigFile) *Model {
	model := &Model{
		cmdchan:       cc,
		newtweetchan:  ntc,
		newtweets_int: make(chan []*Tweet, 10),
		users:         users,
		cur_user:      0,
		lookupchan:    lc,
		tweet_map:     map[int64]*Tweet{},
		uiactionchan:  uac,
		ignored_users: []string{},
	}

	if cfg != nil {
		if default_user, err := cfg.GetString("default", "default_user"); err == nil {
			for idx, _ := range model.users {
				if model.users[idx].User == default_user {
					model.cur_user = uint(idx)
					break
				}
			}
		}

		if ign, err := cfg.GetString("default", "ignore_incoming"); err == nil {
			model.ignored_users = strings.Split(ign, " ")
		}
	}

	userlist := []string{}
	for _, u := range model.users {
		userlist = append(userlist, u.User)
	}
	model.uiactionchan <- ActionSetUserList{Id: model.cur_user, Users: userlist}

	return model
}

func (m *Model) Run() {
	m.last_id = int64(0)

	new_tweets := make(chan []*Tweet, 10)

	go StartUserStreams(m.users, new_tweets, m.uiactionchan, m.ignored_users)

	go func() {
		for {
			if config, err := m.users[m.cur_user].Tapi.Configuration(); err == nil {
				log.Printf("Twitter config data: %v", config)
				if config.Short_url_length != nil {
					m.uiactionchan <- ActionSetURLLength(*config.Short_url_length)
				}
			} else {
				log.Printf("reading Twitter config data failed: %v", err)
			}
			time.Sleep(86400e9)
		}
	}()

	for {
		select {
		case cmd := <-m.cmdchan:
			m.HandleCommand(cmd)
		case req := <-m.lookupchan:
			tweet := m.tweet_map[req.Status_id]
			req.Reply <- tweet
			close(req.Reply)
		case tweets := <-m.newtweets_int:
			for _, t := range tweets {
				m.tweet_map[*t.Id] = t
			}
			m.tweets = append(tweets, m.tweets...)
			if len(tweets) > 0 {
				m.newtweetchan <- tweets
			}
		case tweets := <-new_tweets:
			log.Printf("received %d tweets", len(tweets))
			unique_tweets := []*Tweet{}
			for _, t := range tweets {
				if _, contained := m.tweet_map[*t.Id]; !contained {
					m.tweet_map[*t.Id] = t
					unique_tweets = append(unique_tweets, t)
				}
			}
			m.tweets = append(unique_tweets, m.tweets...)
			log.Printf("received %d tweets (%d unique)", len(tweets), len(unique_tweets))
			if len(unique_tweets) > 0 {
				m.newtweetchan <- unique_tweets
			}
		}
	}
}

func (m *Model) HandleCommand(cmd interface{}) {
	switch v := cmd.(type) {
	case CmdUpdate:
		go func(cur_user uint) {
			if newtweet, err := m.users[cur_user].Tapi.Update(Tweet(v)); err == nil {
				m.newtweets_int <- []*Tweet{newtweet}
				m.uiactionchan <- ActionShowMsg("")
			} else {
				m.uiactionchan <- ActionShowMsg("Posting tweet failed: " + err.Error())
			}
		}(m.cur_user)
	case CmdRetweet:
		go func(cur_user uint) {
			if newtweet, err := m.users[cur_user].Tapi.Retweet(Tweet(v)); err == nil {
				m.newtweets_int <- []*Tweet{newtweet}
				m.uiactionchan <- ActionShowMsg("")
			} else {
				m.uiactionchan <- ActionShowMsg("Retweeting failed:" + err.Error())
			}
		}(m.cur_user)
	case CmdFavorite:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.Favorite(Tweet(v)); err == nil {
				m.uiactionchan <- ActionShowMsg("")
			} else {
				m.uiactionchan <- ActionShowMsg("Favoriting tweet failed: " + err.Error())
			}
		}(m.cur_user)
	case CmdFollow:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.Follow(string(v)); err == nil {
				m.uiactionchan <- ActionShowMsg("")
			} else {
				m.uiactionchan <- ActionShowMsg("Following " + string(v) + " failed: " + err.Error())
			}
		}(m.cur_user)
	case CmdUnfollow:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.Unfollow(TwitterUser(v)); err == nil {
				m.uiactionchan <- ActionShowMsg("")
			} else {
				m.uiactionchan <- ActionShowMsg("Unfollowing " + *TwitterUser(v).Screen_name + " failed: " + err.Error())
			}
		}(m.cur_user)
	case CmdDestroyTweet:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.DestroyTweet(Tweet(v)); err == nil {
				m.uiactionchan <- ActionShowMsg("")
			} else {
				m.uiactionchan <- ActionShowMsg("Deleting tweet failed: " + err.Error())
			}
		}(m.cur_user)
	case CmdSetCurUser:
		m.cur_user = uint(v)
	}
}

type TweetPtrSlice []*Tweet

func (s TweetPtrSlice) Len() int {
	return len(s)
}

func (s TweetPtrSlice) Less(i, j int) bool {
	return *s[j].Id <= *s[i].Id
}

func (s TweetPtrSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func is_ignored(u string, ignored []string) bool {
	for _, s := range ignored {
		if u == s {
			return true
		}
	}
	return false
}

func StartUserStreams(users []UserTwitterAPITuple, new_tweets chan<- []*Tweet, uiactions chan<- interface{}, ignored []string) {
	initial_tweets := []*Tweet{}
	hometl_tweets := make(chan []*Tweet, len(users))

	user_count := 0
	for i, _ := range users {

		if is_ignored(users[i].User, ignored) {
			continue
		}

		user_count++

		go func(i int) {
			if home_tl, err := users[i].Tapi.HomeTimeline(50, 0); err == nil {
				hometl_tweets <- home_tl.Tweets
			} else {
				hometl_tweets <- []*Tweet{}
			}
		}(i)
	}

	for i := 0; i < user_count; i++ {
		tweets := <-hometl_tweets
		initial_tweets = append(initial_tweets, tweets...)
	}

	sort.Sort(TweetPtrSlice(initial_tweets))

	ids := make(map[string]bool, len(initial_tweets))
	unique_tweets := []*Tweet{}
	for _, t := range initial_tweets {
		if _, present := ids[*t.Id_str]; !present {
			ids[*t.Id_str] = true
			unique_tweets = append(unique_tweets, t)
		}
	}

	log.Printf("got %d tweets upfront", len(unique_tweets))
	if len(unique_tweets) > 50 {
		unique_tweets = unique_tweets[0:50]
	}

	new_tweets <- unique_tweets

	for _, u := range users {
		if !is_ignored(u.User, ignored) {
			go u.Tapi.UserStream(new_tweets, uiactions)
		}
	}
}
