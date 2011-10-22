package main

import (
	"bytes"
	"fmt"
	"strconv"
	"os"
	"html"
	"utf8"
	"log"
	stfl "github.com/akrennmair/go-stfl"
	goconf "goconf.googlecode.com/hg"
)

type UserInterface struct {
	form                  *stfl.Form
	actionchan            chan UserInterfaceAction
	tweetchan             chan []*Tweet
	cmdchan               chan TwitterCommand
	lookupchan            chan TweetRequest
	in_reply_to_status_id int64
}

type ActionId int

const (
	RESET_LAST_LINE ActionId = iota
	RAW_INPUT
	DELETE_TWEET
	SHOW_MSG
	KEY_PRESS
)

type UserInterfaceAction struct {
	Action ActionId
	Args   []string
}

func NewUserInterface(cc chan TwitterCommand, tc chan []*Tweet, lc chan TweetRequest, uac chan UserInterfaceAction, cfg *goconf.ConfigFile) *UserInterface {
	stfl.Init()
	ui := &UserInterface{
		form:                  stfl.Create(`vbox[root]
  hbox
    .expand:0
    @style_normal:bg=blue,fg=white,attr=bold
    label text[program]:"" .expand:h
  vbox
    @style_1_normal:fg=yellow,bg=red,attr=bold
    .expand:vh
    list[tweets]
      style_focus[listfocus]:fg=yellow,bg=blue,attr=bold
      .expand:vh
      pos[tweetpos]:0
      pos_name[status_id]:
  vbox
    .expand:0
    .display:1
    label text[infoline]:">> " .expand:h style_normal:bg=blue,fg=yellow,attr=bold
    label text[shorthelp]:"q:Quit ENTER:New Tweet ^R:Retweet r:Reply" .expand:h style_normal:bg=blue,fg=white,attr=bold
  hbox[lastline]
    .expand:0
    label text[msg]:"" .expand:h
`),
		actionchan:            uac,
		tweetchan:             tc,
		cmdchan:               cc,
		in_reply_to_status_id: 0,
		lookupchan:            lc,
	}
	ui.form.Set("program", PROGRAM_NAME + " " + PROGRAM_VERSION)
	return ui
}

func (ui *UserInterface) GetActionChannel() chan UserInterfaceAction {
	return ui.actionchan
}

func (ui *UserInterface) Run() {
	for {
		select {
		case newtweets := <-ui.tweetchan:
			str := formatTweets(newtweets)
			ui.form.Modify("tweets", "insert_inner", str)
			ui.IncrementPosition(len(newtweets))
			ui.UpdateInfoLine()
			ui.form.Run(-1)
		case action := <-ui.actionchan:
			ui.HandleAction(action)
		}
	}
}

func (ui *UserInterface) HandleAction(action UserInterfaceAction) {
	switch action.Action {
	case RESET_LAST_LINE:
		ui.ResetLastLine()
	case RAW_INPUT:
		input := action.Args[0]
		ui.HandleRawInput(input)
	case DELETE_TWEET:
		ui.form.Modify(action.Args[0], "delete", "")
		if current_status_id, err := strconv.Atoi64(ui.form.Get("status_id")); err == nil {
			delete_id, _ := strconv.Atoi64(action.Args[0])
			if delete_id > current_status_id {
				ui.IncrementPosition(-1)
			}
		}
		ui.form.Run(-1)
	case SHOW_MSG:
		ui.form.Set("msg", action.Args[0])
		ui.form.Run(-1)
	case KEY_PRESS:
		ui.UpdateInfoLine()
		ui.UpdateRemaining()
		ui.form.Set("msg", "")
		ui.form.Run(-1)
	}
}

func (ui *UserInterface) ResetLastLine() {
	ui.form.Modify("lastline", "replace", "{hbox[lastline] .expand:0 {label text[msg]:\"\" .expand:h}}")
}

func(ui *UserInterface) UpdateRemaining() {
	if ui.form.GetFocus() == "tweetinput" {
		text := ui.form.Get("inputfield")
		rem_len := 140 - utf8.RuneCountInString(text)
		ui.form.Set("remaining", fmt.Sprintf("%4d ", rem_len))
		if rem_len > 15 {
			ui.form.Set("remaining_style", "fg=white,attr=bold")
		} else if rem_len >= 0 {
			ui.form.Set("remaining_style", "fg=yellow,attr=bold")
		} else if rem_len < 0 {
			ui.form.Set("remaining_style", "fg=white,bg=red,attr=bold")
		}
	}
}

func (ui *UserInterface) UpdateInfoLine() {
	status_id, err := strconv.Atoi64(ui.form.Get("status_id"))
	if err != nil {
		return
	}

	tweet := ui.LookupTweet(status_id)
	if tweet != nil {
		var screen_name, real_name, location, posttime string
		if tweet.User != nil {
			if tweet.User.Screen_name != nil {
				screen_name = *tweet.User.Screen_name
			}
			if tweet.User.Name != nil {
				real_name = *tweet.User.Name
			}
			if tweet.User.Location != nil && *tweet.User.Location != "" {
				location = " - "+*tweet.User.Location
			}
		}
		if tweet.Created_at != nil {
			posttime = tweet.RelativeCreatedAt()
		}
		infoline := fmt.Sprintf(">> @%s (%s)%s | posted %s | https://twitter.com/%s/statuses/%d", screen_name, real_name, location, posttime, screen_name, status_id)
		ui.form.Set("infoline", infoline)
	}
}

func (ui *UserInterface) HandleRawInput(input string) {
	switch input {
	case "ENTER":
		ui.SetInputField("Tweet: ", "", "end-input")
	case "r":
		var err os.Error
		ui.in_reply_to_status_id, err = strconv.Atoi64(ui.form.Get("status_id"))
		if err != nil {
			log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
			ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Error: couldn't determine status ID! (BUG?)"}}
			break
		}
		tweet := ui.LookupTweet(ui.in_reply_to_status_id)
		if tweet != nil {
			ui.SetInputField("Reply: ", "@"+*tweet.User.Screen_name+" ","end-input")
		} else {
			log.Printf("tweet lookup for %d failed\n", ui.in_reply_to_status_id)
			ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Error: tweet lookup by status ID failed! (BUG?)"}}
		}
	case "^R":
		status_id, err := strconv.Atoi64(ui.form.Get("status_id"))
		if err != nil {
			log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
			ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Error: couldn't determine status ID! (BUG?)"}}
			break
		}
		status_id_ptr := new(int64)
		*status_id_ptr = status_id
		ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Retweeting..."}}
		ui.cmdchan <- TwitterCommand{Cmd: RETWEET, Data: Tweet{Id: status_id_ptr}}
	case "^F":
		status_id, err := strconv.Atoi64(ui.form.Get("status_id"))
		if err != nil {
			log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
			ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Error: couldn't determine status ID! (BUG?)"}}
			break
		}
		status_id_ptr := new(int64)
		*status_id_ptr = status_id
		ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Favoriting..."}}
		ui.cmdchan <- TwitterCommand{Cmd: FAVORITE, Data: Tweet{Id: status_id_ptr}}
	case "^O":
		if ui.form.GetFocus() == "tweetinput" {
			text := ui.form.Get("inputfield")
			text = FindURLs(text, ShortenURL)
			ui.form.Set("inputfield", text)
		}
	case "end-input":
		tweet_text := new(string)
		*tweet_text = ui.form.Get("inputfield")
		if len(*tweet_text) > 0 {
			t := Tweet{Text: tweet_text}
			if ui.in_reply_to_status_id != 0 {
				t.In_reply_to_status_id = new(int64)
				*t.In_reply_to_status_id = ui.in_reply_to_status_id
				ui.in_reply_to_status_id = int64(0)
			}
			ui.actionchan <- UserInterfaceAction{SHOW_MSG, []string{"Posting tweet..."}}
			ui.cmdchan <- TwitterCommand{Cmd: UPDATE, Data: t}
		}
		ui.ResetLastLine()
	case "cancel-input":
		ui.ResetLastLine()
	}
	ui.form.Run(-1)
}

func (ui *UserInterface) LookupTweet(status_id int64) *Tweet {
	reply := make(chan *Tweet)
	req := TweetRequest{Status_id: status_id, Reply: reply}
	ui.lookupchan <- req
	return <-reply
}

func (ui *UserInterface) InputLoop() {
	event := ""
	for event != "q" {
		event = ui.form.Run(0)
		if event != "" {
			if event == "^L" {
				stfl.Reset()
			} else {
				ui.actionchan <- UserInterfaceAction{RAW_INPUT, []string{event}}
			}
		} else {
			ui.actionchan <- UserInterfaceAction{Action:KEY_PRESS}
		}
	}
	stfl.Reset()
}

func (ui *UserInterface) SetInputField(prompt, deftext, endevent string) {
	last_line_text := "{hbox[lastline] .expand:0 {label .tie:r .expand:0 text[remaining]:\"\" style_normal[remaining_style]:fg=white}{label .expand:0 text:\"| \"}{label .expand:0 text[prompt]:" + stfl.Quote(prompt) + "}{!input[tweetinput] on_ESC:cancel-input on_ENTER:" + endevent + " modal:1 .expand:h text[inputfield]:" + stfl.Quote(deftext) + " pos[inputpos]:" + strconv.Itoa(utf8.RuneCountInString(deftext)) + "}}"

	ui.form.Modify("lastline", "replace", last_line_text)
	//ui.form.SetFocus("tweetinput")
	ui.UpdateRemaining()
}

func formatTweets(tweets []*Tweet) string {
	buf := bytes.NewBufferString("{list")

	for _, t := range tweets {
		tweetline := fmt.Sprintf("[%16s] %s", "@"+*t.User.Screen_name, *t.Text)
		buf.WriteString(fmt.Sprintf("{listitem[%d] text:%v}", *t.Id, stfl.Quote(html.UnescapeString(tweetline))))
	}

	buf.WriteString("}")
	return string(buf.Bytes())
}

func (ui *UserInterface) IncrementPosition(size int) {
	oldpos, err := strconv.Atoi(ui.form.Get("tweetpos"))
	if err != nil {
		return
	}
	ui.form.Set("tweetpos", fmt.Sprintf("%d", oldpos + size))
}

