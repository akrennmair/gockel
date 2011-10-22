package main

import (
	goconf "goconf.googlecode.com/hg"
)

type Model struct {
	cmdchan       chan TwitterCommand
	newtweetchan  chan []*Tweet
	newtweets_int chan []*Tweet
	tapi          *TwitterAPI
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
)

type TwitterCommand struct {
	Cmd CmdId
	Data Tweet
}

func NewModel(t *TwitterAPI, cc chan TwitterCommand, ntc chan []*Tweet, lc chan TweetRequest, uac chan UserInterfaceAction, cfg *goconf.ConfigFile) *Model {
	model := &Model{
		cmdchan:       cc,
		newtweetchan:  ntc,
		newtweets_int: make(chan []*Tweet, 10),
		tapi:          t,
		lookupchan:    lc,
		tweet_map:     map[int64]*Tweet{},
		uiactionchan:  uac,
	}

	return model
}

func (m *Model) Run() {
	m.last_id = int64(0)

	new_tweets := make(chan []*Tweet, 10)

	// pre-fill with 50 latest items from home timeline
	if home_tl, err := m.tapi.HomeTimeline(50, 0); err == nil {
		if len(home_tl.Tweets) > 0 {
			new_tweets <-home_tl.Tweets
		}
	}

	// then start userstream
	go m.tapi.UserStream(new_tweets, m.uiactionchan)

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
			for _, t := range tweets {
				m.tweet_map[*t.Id] = t
			}
			m.tweets = append(tweets, m.tweets...)
			m.newtweetchan <- tweets
		}
	}
}


func (m *Model) HandleCommand(cmd TwitterCommand) {
	switch cmd.Cmd {
	case UPDATE:
		go func() {
			if newtweet, err := m.tapi.Update(cmd.Data); err == nil {
				m.newtweets_int <- []*Tweet{newtweet}
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Posting tweet failed: " + err.String()}}
			}
		}()
	case RETWEET:
		go func() {
			if newtweet, err := m.tapi.Retweet(cmd.Data); err == nil {
				m.newtweets_int <- []*Tweet{newtweet}
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Retweeting failed:" + err.String()}}
			}
		}()
	case FAVORITE:
		go func() {
			if err := m.tapi.Favorite(cmd.Data); err == nil {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{""}}
			} else {
				m.uiactionchan <- UserInterfaceAction{SHOW_MSG, []string{"Favoriting tweet failed: " + err.String()}}
			}
		}()
	}
}

