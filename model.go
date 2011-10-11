package main

import (
	"time"
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

	new_tweets := make(chan *Timeline, 10)
	go func() {
		ticker := make(chan int)
		go Ticker(ticker, 20e9)
		for {
			select {
			case <-ticker:
				home_tl, err := m.tapi.HomeTimeline(50, m.last_id)
				if err == nil && len(home_tl.Tweets) > 0 && home_tl.Tweets[0].Id != nil {
					m.last_id = *home_tl.Tweets[0].Id
					new_tweets <-home_tl
				}
			}
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
		case home_tl := <-new_tweets:
			m.UpdateRateLimit()
			for _, t := range home_tl.Tweets {
				m.tweet_map[*t.Id] = t
			}
			m.tweets = append(home_tl.Tweets, m.tweets...)
			m.newtweetchan <- home_tl.Tweets
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
			m.newtweetchan <- []*Tweet{newtweet}
		}
		m.UpdateRateLimit()
	case RETWEET:
		if newtweet, err := m.tapi.Retweet(cmd.Data); err == nil {
			m.tweet_map[*newtweet.Id] = newtweet
			//m.last_id = *newtweet.Id // how does this react?
			m.tweets = append([]*Tweet{newtweet}, m.tweets...)
			m.newtweetchan <- []*Tweet{newtweet}
		}
		m.UpdateRateLimit()
		// TODO: add more commands here
	}
}

func (m *Model) UpdateRateLimit() {
	rem, limit, reset := m.tapi.GetRateLimit()
	m.uiactionchan <- UserInterfaceAction{Action: UPDATE_RATELIMIT, Args: []string{ strconv.Uitoa(rem), strconv.Uitoa(limit), strconv.Itoa64(reset) }}
}

func Ticker(tickchan chan int, ns int64) {
	for {
		tickchan <- 1
		time.Sleep(ns)
	}
}
