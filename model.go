package main

import (
	"time"
)

type Model struct {
	updatechan chan string
	newtweetchan chan []Tweet
	tapi *TwitterAPI
}

func NewModel(t *TwitterAPI) *Model {
	model := &Model{
		updatechan: make(chan string, 1),
		newtweetchan: make(chan []Tweet, 1),
		tapi: t,
	}

	return model
}

func(m *Model) GetUpdateChannel() chan string {
	return m.updatechan
}

func(m *Model) GetNewTweetChannel() chan []Tweet {
	return m.newtweetchan
}

func(m *Model) Run() {
	ticker := make(chan int, 1)
	go Ticker(ticker, 20e9)

	last_id := int64(0)

	for {
		select {
		case tweet := <-m.updatechan:
			m.tapi.Update(tweet)
		case <-ticker:
			home_tl, err := m.tapi.HomeTimeline(50, last_id)

			if err != nil {
				//TODO: signal error
			} else {
				if len(home_tl.Tweets) > 0 {
					m.newtweetchan <- home_tl.Tweets
					if home_tl.Tweets[0].Id != nil {
						last_id = *home_tl.Tweets[0].Id
					}
				}
			}
		}
	}
}

func Ticker(tickchan chan int, ns int64) {
	for {
		tickchan <-1
		time.Sleep(ns)
	}
}
