package main

import (
	"bytes"
	"fmt"
	"strconv"
	"os"
	"html"
	"utf8"
	"log"
	"regexp"
	"strings"
	stfl "github.com/akrennmair/go-stfl"
	goconf "goconf.googlecode.com/hg"
)

type UserInterface struct {
	form                  *stfl.Form
	actionchan            chan interface{}
	tweetchan             <-chan []*Tweet
	cmdchan               chan<- interface{}
	lookupchan            chan<- TweetRequest
	in_reply_to_status_id int64
	cfg                   *goconf.ConfigFile
	highlight_rx          []*regexp.Regexp
	users                 []string
	cur_user              uint
	short_url_length      int
	confirm_quit          bool
}

type ActionId int

type ActionResetLastLine struct{}

type ActionRawInput string

type ActionDeleteTweet int64

type ActionShowMsg string

type ActionKeyPress struct{}

type ActionSetUserList struct {
	Id    uint
	Users []string
}

type ActionSetURLLength uint

func NewUserInterface(cc chan<- interface{}, tc <-chan []*Tweet, lc chan<- TweetRequest, uac chan interface{}, cfg *goconf.ConfigFile) *UserInterface {
	stfl.Init()
	ui := &UserInterface{
		form: stfl.Create(`vbox[root]
  @style_normal[style_background]:
  hbox
    .expand:0
    label text[userlist]:"" .expand:h style_normal[style_userlist]:bg=blue,fg=yellow,attr=bold richtext:1 style_s_normal[style_userlist_active]:bg=white,fg=blue,attr=bold
  vbox
    .expand:vh
    list[tweets]
      ** just a place holder to be filled by constructTweetList()
  vbox
    .expand:0
    .display:1
    hbox
      @style_normal[style_infotext]:bg=blue,fg=yellow,attr=bold
      label text[infoline]:">> " .expand:h
      label text[program]:"" .expand:0
    label text[shorthelp]:"q:Quit ENTER:New Tweet ^R:Retweet r:Reply R:Public Reply ^F:Favorite D:Delete" .expand:h style_normal[style_shorthelp]:bg=blue,fg=white,attr=bold
  hbox[lastline]
    .expand:0
    label text[msg]:"" .expand:h
`),
		actionchan:            uac,
		tweetchan:             tc,
		cmdchan:               cc,
		in_reply_to_status_id: 0,
		lookupchan:            lc,
		cfg:                   cfg,
		highlight_rx:          []*regexp.Regexp{},
		users:                 []string{},
		cur_user:              0,
		confirm_quit:          false,
	}
	ui.constructTweetList()
	ui.setColors()
	ui.form.Set("program", " "+PROGRAM_NAME+" "+PROGRAM_VERSION)

	if ui.cfg != nil {
		if confirm_quit, err := ui.cfg.GetBool("default", "confirm_quit"); err == nil {
			ui.confirm_quit = confirm_quit
		}
	}

	return ui
}

func (ui *UserInterface) setColors() {
	if ui.cfg == nil {
		return
	}

	for _, elem := range []string{"shorthelp", "infotext", "listfocus", "listnormal", "background", "input", "userlist", "userlist_active"} {
		if value, err := ui.cfg.GetString("colors", elem); err == nil && value != "" {
			// TODO: check whether value is syntactically valid
			ui.form.Set("style_"+elem, value)
		}
	}
}

func (ui *UserInterface) constructTweetList() {
	buf := bytes.NewBufferString("{list[tweets] style_focus[style_listfocus]:fg=yellow,bg=blue,attr=bold style_normal[style_listnormal]: .expand:vh pos[tweetpos]:0 pos_name[status_id]: ")

	log.Printf("constructing actual tweet list")

	count := 0

	if ui.cfg != nil {
		for _, section := range ui.cfg.GetSections() {
			if !strings.HasPrefix(section, "highlight") {
				continue
			}

			attr_str, err := ui.cfg.GetString(section, "attributes")
			if err != nil {
				continue
			}

			rx, err := ui.cfg.GetString(section, "regex")
			if err != nil {
				continue
			}

			if rx[0:1] == "/" && rx[len(rx)-1:] == "/" {
				rx = rx[1 : len(rx)-1]
			}

			compiled_rx, err := regexp.Compile(rx)

			if err != nil {
				log.Printf("regex %s failed to compile: %v", rx, err)
			}

			ui.highlight_rx = append(ui.highlight_rx, compiled_rx)

			log.Printf("configured regex '%s' with attributes %s at position %d", rx, attr_str, count)

			buf.WriteString(fmt.Sprintf("@style_%d_normal:%s @style_%d_focus:%s ", count, attr_str, count, attr_str))

			count++
		}
	}

	buf.WriteString(" richtext:1}")
	ui.form.Modify("tweets", "replace", string(buf.Bytes()))
}

func (ui *UserInterface) GetActionChannel() chan interface{} {
	return ui.actionchan
}

func (ui *UserInterface) Run() {
	for {
		select {
		case newtweets := <-ui.tweetchan:
			log.Printf("received %d tweets", len(newtweets))
			ui.addTweets(newtweets)
			ui.IncrementPosition(len(newtweets))
			ui.UpdateInfoLine()
			ui.form.Run(-1)
		case action := <-ui.actionchan:
			ui.HandleAction(action)
		}
	}
}

func (ui *UserInterface) HandleAction(action interface{}) {
	switch v := action.(type) {
	case ActionResetLastLine:
		ui.ResetLastLine()
	case ActionRawInput:
		ui.HandleRawInput(string(v))
	case ActionDeleteTweet:
		delete_id := int64(v)
		ui.form.Modify(strconv.Itoa64(delete_id), "delete", "")
		if current_status_id, err := strconv.Atoi64(ui.form.Get("status_id")); err == nil {
			if delete_id > current_status_id {
				ui.IncrementPosition(-1)
			}
		}
		ui.form.Run(-1)
	case ActionShowMsg:
		ui.form.Set("msg", string(v))
		ui.form.Run(-1)
	case ActionKeyPress:
		ui.UpdateInfoLine()
		ui.UpdateRemaining()
		ui.form.Set("msg", "")
		ui.form.Run(-1)
	case ActionSetUserList:
		ui.cur_user = v.Id
		ui.users = v.Users
		ui.UpdateUserList()
		ui.form.Run(-1)
	case ActionSetURLLength:
		ui.short_url_length = int(v)
	}
}

func (ui *UserInterface) UpdateUserList() {
	buf := bytes.NewBufferString("")

	for i, u := range ui.users {
		if i == int(ui.cur_user) {
			buf.WriteString("<s>")
		}
		buf.WriteString(fmt.Sprintf("%d:%s", i+1, strings.Replace(u, "<", "<>", -1)))
		if i == int(ui.cur_user) {
			buf.WriteString("*</>")
		}
		buf.WriteString(" ")
	}

	ui.form.Set("userlist", string(buf.Bytes()))
}

func (ui *UserInterface) ResetLastLine() {
	ui.form.Modify("lastline", "replace", "{hbox[lastline] .expand:0 {label text[msg]:\"\" .expand:h}}")
}

func (ui *UserInterface) UpdateRemaining() {
	if ui.form.GetFocus() == "tweetinput" {
		text := ui.form.Get("inputfield")
		rem_len := 140 - utf8.RuneCountInString(text)
		if ui.short_url_length > 0 {
			FindURLs(text, func(url string) string {
				rem_len += len(url)
				rem_len -= ui.short_url_length
				return url
			})
		}
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
				location = " - " + *tweet.User.Location
			}
		}
		if tweet.Created_at != nil {
			posttime = tweet.RelativeCreatedAt()
		}
		infoline := fmt.Sprintf(">> @%s (%s)%s | posted %s | https://twitter.com/%s/statuses/%d", screen_name, real_name, location, posttime, screen_name, status_id)
		ui.form.Set("infoline", infoline)
	}
}

func (ui *UserInterface) StartReply(public bool) {
	var err os.Error
	ui.in_reply_to_status_id, err = strconv.Atoi64(ui.form.Get("status_id"))
	if err != nil {
		log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
		ui.actionchan <- ActionShowMsg("Error: couldn't determine status ID! (BUG?)")
		return
	}
	tweet := ui.LookupTweet(ui.in_reply_to_status_id)
	if tweet != nil {
		default_text := "@" + *tweet.User.Screen_name + " "
		if public {
			default_text = "." + default_text
		}
		ui.SetInputField("Reply: ", default_text, "end-input", true)
	} else {
		log.Printf("tweet lookup for %d failed\n", ui.in_reply_to_status_id)
		ui.actionchan <- ActionShowMsg("Error: tweet lookup by status ID failed! (BUG?)")
	}
}

func (ui *UserInterface) HandleRawInput(input string) {
	switch input {
	case "ENTER":
		ui.SetInputField("Tweet: ", "", "end-input", true)
	case "R":
		ui.StartReply(true)
	case "r":
		ui.StartReply(false)
	case "^R":
		status_id, err := strconv.Atoi64(ui.form.Get("status_id"))
		if err != nil {
			log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
			ui.actionchan <- ActionShowMsg("Error: couldn't determine status ID! (BUG?)")
			break
		}
		status_id_ptr := new(int64)
		*status_id_ptr = status_id
		ui.actionchan <- ActionShowMsg("Retweeting...")
		ui.cmdchan <- CmdRetweet(Tweet{Id: status_id_ptr})
	case "^E":
		rt_status_id, err := strconv.Atoi64(ui.form.Get("status_id"))
		if err != nil {
			log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
			ui.actionchan <- ActionShowMsg("Error: couldn't determine status ID! (BUG?)")
			break
		}
		tweet := ui.LookupTweet(rt_status_id)
		if tweet != nil {
			rt_text := "RT @" + *tweet.User.Screen_name + ": " + *tweet.Text
			ui.SetInputField("Tweet: ", rt_text, "end-input", true)
		} else {
			log.Printf("tweet lookup for %d failed\n", ui.in_reply_to_status_id)
			ui.actionchan <- ActionShowMsg("Error: tweet lookup by status ID failed! (BUG?)")
		}
	case "^F":
		status_id, err := strconv.Atoi64(ui.form.Get("status_id"))
		if err != nil {
			log.Printf("conversion of %s failed: %v", ui.form.Get("status_id"), err)
			ui.actionchan <- ActionShowMsg("Error: couldn't determine status ID! (BUG?)")
			break
		}
		status_id_ptr := new(int64)
		*status_id_ptr = status_id
		ui.actionchan <- ActionShowMsg("Favoriting...")
		ui.cmdchan <- CmdFavorite(Tweet{Id: status_id_ptr})
	case "F":
		ui.SetInputField("Follow: ", "", "end-input-follow", false)
	case "end-input-follow":
		screen_name := ui.form.Get("inputfield")
		ui.actionchan <- ActionShowMsg("Following " + screen_name + "...")
		ui.cmdchan <- CmdFollow(screen_name)
		ui.ResetLastLine()
	case "U":
		if status_id, err := strconv.Atoi64(ui.form.Get("status_id")); err == nil {
			if tweet := ui.LookupTweet(status_id); tweet != nil {
				ui.actionchan <- ActionShowMsg("Unfollowing " + *tweet.User.Screen_name + "...")
				ui.cmdchan <- CmdUnfollow(*tweet.User)
			}
		}
	case "D":
		if status_id, err := strconv.Atoi64(ui.form.Get("status_id")); err == nil {
			if tweet := ui.LookupTweet(status_id); tweet != nil {
				ui.actionchan <- ActionShowMsg("Deleting tweet...")
				ui.cmdchan <- CmdDestroyTweet(*tweet)
			}
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		pos, _ := strconv.Atoi(input)
		pos -= 1
		if pos < len(ui.users) {
			ui.cmdchan <- CmdSetCurUser(pos)
			ui.cur_user = uint(pos)
			ui.UpdateUserList()
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
			ui.actionchan <- ActionShowMsg("Posting tweet...")
			ui.cmdchan <- CmdUpdate(t)
		}
		ui.ResetLastLine()
	case "cancel-input":
		ui.ResetLastLine()
		ui.in_reply_to_status_id = 0
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
	for {
		if event == "q" {
			if !ui.confirm_quit {
				break
			} else {
				ui.actionchan <- ActionShowMsg("Quit " + PROGRAM_NAME + " (y/[n])?")
				event = ui.form.Run(0)
				if event == "y" {
					break
				}
				ui.actionchan <- ActionShowMsg("")
				event = ""
			}
		}
		event = ui.form.Run(0)
		if event != "" {
			if event == "^L" {
				stfl.Reset()
			} else {
				ui.actionchan <- ActionRawInput(event)
			}
		} else {
			ui.actionchan <- ActionKeyPress{}
		}
	}
	stfl.Reset()
}

func (ui *UserInterface) SetInputField(prompt, deftext, endevent string, show_rem bool) {
	pos := strconv.Itoa(utf8.RuneCountInString(deftext))
	buf := bytes.NewBufferString("{hbox[lastline] @style_normal[style_input]: .expand:0 ")
	if show_rem {
		buf.WriteString("{label .tie:r .expand:0 text[remaining]:\"\" style_normal[remaining_style]:fg=white}{label .expand:0 text:\"| \"}")
	}
	buf.WriteString("{label .expand:0 text[prompt]:")
	buf.WriteString(stfl.Quote(prompt))
	buf.WriteString("}{!input[tweetinput] on_ESC:cancel-input on_ENTER:")
	buf.WriteString(endevent)
	buf.WriteString(" modal:1 .expand:h text[inputfield]:")
	buf.WriteString(stfl.Quote(deftext))
	buf.WriteString(" pos[inputpos]:0 offset:0}}")

	ui.form.Modify("lastline", "replace", string(buf.Bytes()))
	ui.form.Run(-1)
	ui.form.Set("inputpos", pos)
	ui.UpdateRemaining()
}

func (ui *UserInterface) addTweets(tweets []*Tweet) {
	buf := bytes.NewBufferString("{list")

	for _, t := range tweets {
		tweetline := fmt.Sprintf("[%16s] %s", "@"+*t.User.Screen_name, html.UnescapeString(strings.Replace(strings.Replace(*t.Text, "\n", " ", -1), "\r", " ", -1)))
		tweetline = strings.Replace(tweetline, "<", "<>", -1)
		tweetline = ui.highlight(tweetline)
		buf.WriteString(fmt.Sprintf("{listitem[%d] text:%v}", *t.Id, stfl.Quote(tweetline)))
	}

	buf.WriteString("}")
	ui.form.Modify("tweets", "insert_inner", string(buf.Bytes()))
}

func (ui *UserInterface) highlight(str string) string {
	for idx, rx := range ui.highlight_rx {
		str = rx.ReplaceAllStringFunc(str, func(s string) string {
			return fmt.Sprintf("<%d>%s</>", idx, s)
		})
	}
	return str
}

func (ui *UserInterface) IncrementPosition(size int) {
	oldpos, err := strconv.Atoi(ui.form.Get("tweetpos"))
	if err != nil {
		return
	}
	ui.form.Set("tweetpos", fmt.Sprintf("%d", oldpos+size))
}
