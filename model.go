package main

import (
	"time"
	"fmt"
	"os"
)

type Model struct {
	cmdchan      chan TwitterCommand
	newtweetchan chan []*Tweet
	tapi         *TwitterAPI
	tweets       []*Tweet
	tweet_map    map[int64]*Tweet
	lookupchan   chan TweetRequest
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

func NewModel(t *TwitterAPI) *Model {
	model := &Model{
		cmdchan:      make(chan TwitterCommand, 1),
		newtweetchan: make(chan []*Tweet, 1),
		tapi:         t,
		lookupchan:   make(chan TweetRequest, 1),
		tweet_map:    map[int64]*Tweet{},
	}

	return model
}

func (m *Model) GetCommandChannel() chan TwitterCommand {
	return m.cmdchan
}

func (m *Model) GetNewTweetChannel() chan []*Tweet {
	return m.newtweetchan
}

func (m *Model) GetLookupChannel() chan TweetRequest {
	return m.lookupchan
}

func (m *Model) Run() {
	ticker := make(chan int, 1)
	go Ticker(ticker, 20e9)

	m.last_id = int64(0)

	for {
		select {
		case cmd := <-m.cmdchan:
			m.HandleCommand(cmd)
		case req := <-m.lookupchan:
			tweet := m.tweet_map[req.Status_id]
			req.Reply <- tweet
			close(req.Reply)
		case <-ticker:
			home_tl, err := m.tapi.HomeTimeline(50, m.last_id)

			if err != nil {
				//TODO: signal error
			} else {
				if len(home_tl.Tweets) > 0 {
					for _, t := range home_tl.Tweets {
						m.tweet_map[*t.Id] = t
					}
					m.tweets = append(home_tl.Tweets, m.tweets...)
					m.newtweetchan <- home_tl.Tweets
					if home_tl.Tweets[0].Id != nil {
						m.last_id = *home_tl.Tweets[0].Id
					}
				}
			}
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
	case RETWEET:
		if newtweet, err := m.tapi.Retweet(cmd.Data); err == nil {
			fmt.Fprintf(os.Stderr, "%v\n", *newtweet)
			m.tweet_map[*newtweet.Id] = newtweet
			//m.last_id = *newtweet.Id // how does this react?
			m.tweets = append([]*Tweet{newtweet}, m.tweets...)
			m.newtweetchan <- []*Tweet{newtweet}
		}
		// TODO: add more commands here
	}
}

func Ticker(tickchan chan int, ns int64) {
	for {
		tickchan <- 1
		time.Sleep(ns)
	}
}
