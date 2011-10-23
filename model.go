package main

import (
	goconf "goconf.googlecode.com/hg"
	"strconv"
	"log"
)

type Model struct {
	cmdchan       <-chan TwitterCommand
	newtweetchan  chan<- []*Tweet
	newtweets_int chan []*Tweet
	users         []UserTwitterAPITuple
	cur_user      uint
	tweets        []*Tweet
	tweet_map     map[int64]*Tweet
	lookupchan    chan TweetRequest
	uiactionchan  chan UserInterfaceAction
	last_id       int64
}

type TweetRequest struct {
	Status_id int64
	Reply     chan *Tweet
}

type CmdId int

const (
	UPDATE CmdId = iota
	RETWEET
	DELETE
	FAVORITE
	FOLLOW
	UNFOLLOW
	SET_CURUSER
)

type TwitterCommand struct {
	Cmd CmdId
	Data Tweet
	Pos uint // for SET_CURUSER
}

func NewModel(users []UserTwitterAPITuple, cc chan TwitterCommand, ntc chan<- []*Tweet, lc chan TweetRequest, uac chan UserInterfaceAction, cfg *goconf.ConfigFile) *Model {
	model := &Model{
		cmdchan:       cc,
		newtweetchan:  ntc,
		newtweets_int: make(chan []*Tweet, 10),
		users:         users,
		cur_user:      0,
		lookupchan:    lc,
		tweet_map:     map[int64]*Tweet{},
		uiactionchan:  uac,
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
	}

	userlist := []string{ strconv.Uitoa(model.cur_user) }
	for _, u := range model.users {
		userlist = append(userlist, u.User)
	}
	model.uiactionchan <- UserInterfaceAction{SET_USERLIST, userlist}

	return model
}

func (m *Model) Run() {
	m.last_id = int64(0)

	new_tweets := make(chan []*Tweet, 10)

	// pre-fill with 50 latest items from home timeline
	// TODO: move this to StartUserStreams, including merge, sort and deduplication
	/*
	if home_tl, err := m.users[m.cur_user].Tapi.HomeTimeline(50, 0); err == nil {
		if len(home_tl.Tweets) > 0 {
			new_tweets <-home_tl.Tweets
		}
	}
	*/

	// then start userstream
	StartUserStreams(m.users, new_tweets, m.uiactionchan)

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
		case tweets := <-new_tweets:
			log.Printf("received %d tweets", len(tweets))
			unique_tweets := []*Tweet{}
			for _, t := range tweets {
				if _, contained := m.tweet_map[*t.Id]; !contained {
					m.tweet_map[*t.Id] = t
					unique_tweets = append(unique_tweets, t)
				}
				m.tweets = append(unique_tweets, m.tweets...)
			}
			m.newtweetchan <- unique_tweets
		}
	}
}


func (m *Model) HandleCommand(cmd TwitterCommand) {
	switch cmd.Cmd {
	case UPDATE:
		go func(cur_user uint) {
			if newtweet, err := m.users[cur_user].Tapi.Update(cmd.Data); err == nil {
				m.newtweets_int <- []*Tweet{newtweet}
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Posting tweet failed: " + err.String()}}
			}
		}(m.cur_user)
	case RETWEET:
		go func(cur_user uint) {
			if newtweet, err := m.users[cur_user].Tapi.Retweet(cmd.Data); err == nil {
				m.newtweets_int <- []*Tweet{newtweet}
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Retweeting failed:" + err.String()}}
			}
		}(m.cur_user)
	case FAVORITE:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.Favorite(cmd.Data); err == nil {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Favoriting tweet failed: " + err.String()}}
			}
		}(m.cur_user)
	case FOLLOW:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.Follow(*cmd.Data.User.Screen_name); err == nil {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Following " + *cmd.Data.User.Screen_name + " failed: " + err.String()}}
			}
		}(m.cur_user)
	case UNFOLLOW:
		go func(cur_user uint) {
			if err := m.users[cur_user].Tapi.Unfollow(*cmd.Data.User); err == nil {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Unfollowing " + *cmd.Data.User.Screen_name + " failed: " + err.String()}}
			}
		}(m.cur_user)
	case SET_CURUSER:
		m.cur_user = cmd.Pos
	}
}

func StartUserStreams(users []UserTwitterAPITuple, new_tweets chan<- []*Tweet, uiactions chan<- UserInterfaceAction) {
	for _, u := range users {
		go u.Tapi.UserStream(new_tweets, uiactions)
	}
}
