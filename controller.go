package main

type Controller struct {
	tweets       []Tweet
	newtweetchan chan []Tweet
	viewchan     chan []Tweet
}

func NewController(ntchan chan []Tweet) *Controller {
	ctrl := &Controller{
		tweets:       []Tweet{},
		newtweetchan: ntchan,
		viewchan:     make(chan []Tweet, 1),
	}
	return ctrl
}

func (ctrl *Controller) GetViewChannel() chan []Tweet {
	return ctrl.viewchan
}

func (ctrl *Controller) Run() {
	for {
		select {
		case newtweets := <-ctrl.newtweetchan:
			ctrl.tweets = append(newtweets, ctrl.tweets...)
			ctrl.viewchan <- newtweets
		}
	}
}
