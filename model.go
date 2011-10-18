package main

import (
	"strconv"
)

type Model struct {
	cmdchan      chan TwitterCommand
	newtweetchan chan []*Tweet
	tapi         *TwitterAPI
	tweets       []*Tweet
	tweet_map    map[int64]*Tweet
	lookupchan   chan TweetRequest
	uiactionchan chan UserInterfaceAction
	last_id      int64
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

func NewModel(t *TwitterAPI, cc chan TwitterCommand, ntc chan []*Tweet, lc chan TweetRequest, uac chan UserInterfaceAction) *Model {
	model := &Model{
		cmdchan:      cc,
		newtweetchan: ntc,
		tapi:         t,
		lookupchan:   lc,
		tweet_map:    map[int64]*Tweet{},
		uiactionchan: uac,
	}

	return model
}

func (m *Model) Run() {
	m.last_id = int64(0)

	new_tweets := make(chan []*Tweet, 10)

	// pre-fill with 50 latest items from home timeline
	home_tl, err := m.tapi.HomeTimeline(50, 0)
	if err == nil && len(home_tl.Tweets) > 0 {
		new_tweets <-home_tl.Tweets
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
		case tweets := <-new_tweets:
			m.UpdateRateLimit()
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
		if newtweet, err := m.tapi.Update(cmd.Data); err == nil {
			m.tweet_map[*newtweet.Id] = newtweet
			m.last_id = *newtweet.Id
			m.tweets = append([]*Tweet{newtweet}, m.tweets...)
		}
		m.UpdateRateLimit()
	case RETWEET:
		if newtweet, err := m.tapi.Retweet(cmd.Data); err == nil {
			m.tweet_map[*newtweet.Id] = newtweet
			//m.last_id = *newtweet.Id // how does this react?
			m.tweets = append([]*Tweet{newtweet}, m.tweets...)
		}
		m.UpdateRateLimit()
		// TODO: add more commands here
	case FAVORITE:
		if err := m.tapi.Favorite(cmd.Data); err != nil {
			// TODO: show error
		}
		m.UpdateRateLimit()
	}
}

func (m *Model) UpdateRateLimit() {
	rem, limit, reset := m.tapi.GetRateLimit()
	m.uiactionchan <- UserInterfaceAction{Action: UPDATE_RATELIMIT, Args: []string{ strconv.Uitoa(rem), strconv.Uitoa(limit), strconv.Itoa64(reset) }}
}

