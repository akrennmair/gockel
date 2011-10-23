package main

import (
	goconf "goconf.googlecode.com/hg"
	"strconv"
	"log"
	"sort"
)

type Model struct {
	cmdchan       <-chan TwitterCommand
	newtweetchan  chan<- []*Tweet
	newtweets_int chan []*Tweet
	users         []UserTwitterAPITuple
	cur_user      uint
	tweets        []*Tweet
	tweet_map     map[int64]*Tweet
	lookupchan    <-chan TweetRequest
	uiactionchan  chan<- UserInterfaceAction
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

func NewModel(users []UserTwitterAPITuple, cc chan TwitterCommand, ntc chan<- []*Tweet, lc <-chan TweetRequest, uac chan<- UserInterfaceAction, cfg *goconf.ConfigFile) *Model {
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

	go StartUserStreams(m.users, new_tweets, m.uiactionchan)

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
			log.Printf("received %d tweets (%d unique)", len(tweets), len(unique_tweets))
			if len(unique_tweets) > 0 {
				m.newtweetchan <- unique_tweets
			}
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

func StartUserStreams(users []UserTwitterAPITuple, new_tweets chan<- []*Tweet, uiactions chan<- UserInterfaceAction) {
	initial_tweets := []*Tweet{}
	hometl_tweets := make(chan []*Tweet, len(users))

	for i, _ := range users {
		go func(i int) {
			if home_tl, err := users[i].Tapi.HomeTimeline(20, 0); err == nil {
				hometl_tweets <-home_tl.Tweets
			} else {
				hometl_tweets <-[]*Tweet{}
			}
		}(i)
	}

	for i:=0;i<len(users);i++ {
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
		go u.Tapi.UserStream(new_tweets, uiactions)
	}
}
