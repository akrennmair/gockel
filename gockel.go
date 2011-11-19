package main

import (
	"fmt"
	"os"
	"os/user"
	"json"
	"io/ioutil"
	"flag"
	"log"
	"strings"
	oauth "github.com/akrennmair/goauth"
	goconf "goconf.googlecode.com/hg"
)

const (
	PROGRAM_NAME    = "gockel"
	PROGRAM_VERSION = "0.2"
	PROGRAM_URL     = "https://github.com/akrennmair/gockel"

	CONSUMER_KEY    = "sDggzGbHbyAfl5fJ87XOCA"
	CONSUMER_SECRET = "MOCQDL7ot7qIxMwYL5x1mMAqiYBYxNTxPWS6tc6hw"
)

type DevNullWriter int

func (w *DevNullWriter) Write(b []byte) (n int, err os.Error) {
	return len(b), nil
}

func main() {
	homedir := getHomeDir()
	if homedir == "" {
		fmt.Printf("Error: unable to determine home directory!\n")
		return
	}

	cfgdir := homedir + "/.gockel"
	os.Mkdir(cfgdir, 0700)

	cfgfilename := cfgdir + "/gockelrc"

	var logfile *string = flag.String("log", "", "logfile")
	var cfgfile *string = flag.String("config", cfgfilename, "configuration file")
	var add *bool = flag.Bool("add", false, "add a new user")

	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	devnull := new(DevNullWriter)
	log.SetOutput(devnull)

	if *logfile != "" {
		if f, err := os.OpenFile(*logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600); err == nil {
			defer f.Close()
			log.SetOutput(f)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: couldn't open logfile for writing: %v\n", err)
		}
	}

	cfg, err := goconf.ReadConfigFile(*cfgfile)
	if err != nil {
		log.Printf("reading configuration file failed: %v", err)
	}

	tapi := NewTwitterAPI(CONSUMER_KEY, CONSUMER_SECRET, cfg)

	if *add {
		if err := AddUser(tapi, cfgdir); err != nil {
			log.Printf("Error while adding user: %v", err)
			fmt.Fprintf(os.Stderr, "Error while adding user: %v\n", err)
		}
		return
	}

	fmt.Printf("Starting %s %s...\n", PROGRAM_NAME, PROGRAM_VERSION)

	var users []UserTwitterAPITuple

	fmt.Printf("Loading user information...")
	for {
		users, err = LoadAccessTokens(cfgdir, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Loading users failed: %v\n", err)
			return
		}
		if len(users) > 0 {
			break
		}
		err = AddUser(tapi, cfgdir)
		if err != nil {
			log.Printf("Error while adding user: %v", err)
			fmt.Fprintf(os.Stderr, "Error while adding user: %v\n", err)
			return
		}
	}

	fmt.Println(" done.")

	log.Printf("loaded %d users:", len(users))
	for _, u := range users {
		log.Printf("user: %s", u.User)
	}

	cmdchan := make(chan TwitterCommand, 1)
	newtweetchan := make(chan []*Tweet, 10)
	lookupchan := make(chan TweetRequest, 1)
	uiactionchan := make(chan UserInterfaceAction, 10)

	model := NewModel(users, cmdchan, newtweetchan, lookupchan, uiactionchan, cfg)
	go model.Run()

	ui := NewUserInterface(cmdchan, newtweetchan, lookupchan, uiactionchan, cfg)
	go ui.Run()

	ui.InputLoop()
}

func SaveAccessToken(at *oauth.AccessToken, cfgdir string, suffix string) os.Error {
	data, marshalerr := json.Marshal(at)
	if marshalerr != nil {
		return marshalerr
	}

	filename := cfgdir + "/access_token.json"
	if suffix != "" {
		filename = filename + "." + suffix
	}

	f, ferr := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if ferr != nil {
		return ferr
	}
	defer f.Close()

	_, werr := f.Write(data)
	if werr != nil {
		return werr
	}

	return nil
}

type UserTwitterAPITuple struct {
	User string
	Tapi *TwitterAPI
}

func LoadAccessTokens(cfgdir string, cfg *goconf.ConfigFile) ([]UserTwitterAPITuple, os.Error) {
	users := []UserTwitterAPITuple{}

	f, err := os.Open(cfgdir)
	if err != nil {
		log.Printf("failed to open %s: %v", cfgdir, err)
		return users, err
	}
	defer f.Close()

	files, err := f.Readdir(-1)
	if err != nil {
		log.Printf("readdir failed: %v", err)
		return users, err
	}

	log.Printf("found %d files in %s", len(files), cfgdir)

	for _, fi := range files {
		log.Printf("file: %s", fi.Name)
		if strings.HasPrefix(fi.Name, "access_token.json") {
			if at, err := LoadAccessToken(cfgdir + "/" + fi.Name); err == nil {
				tapi := NewTwitterAPI(CONSUMER_KEY, CONSUMER_SECRET, cfg)
				tapi.SetAccessToken(at)
				if user, err := tapi.VerifyCredentials(); err == nil {
					users = append(users, UserTwitterAPITuple{User: *user.Screen_name, Tapi: tapi})
				}
			} else {
				log.Printf("loading access token from %s failed: %v", fi.Name, err)
			}
		}
	}

	return users, nil
}

func LoadAccessToken(file string) (*oauth.AccessToken, os.Error) {
	f, ferr := os.Open(file)
	if ferr != nil {
		return nil, ferr
	}
	defer f.Close()

	data, readerr := ioutil.ReadAll(f)
	if readerr != nil {
		return nil, readerr
	}

	at := &oauth.AccessToken{}

	err := json.Unmarshal(data, at)
	if err != nil {
		return nil, err
	}

	return at, nil
}

func getHomeDir() string {
	// first try $HOME
	homedir := os.Getenv("HOME")
	if homedir != "" {
		return homedir
	}

	// then try to lookup by username from $USER
	u_name, err := user.Lookup(os.Getenv("USER"))
	if err == nil {
		return u_name.HomeDir
	}

	// then try lookup by uid
	u_uid, err := user.LookupId(os.Getuid())
	if err == nil {
		return u_uid.HomeDir
	}

	return ""
}

func AddUser(tapi *TwitterAPI, cfgdir string) os.Error {
	auth_url, err := tapi.GetRequestAuthorizationURL()
	if err != nil {
		return err
	}

	var pin string
	fmt.Printf("%s doesn't yet have information how to access your Twitter account.\n", PROGRAM_NAME)
	fmt.Println("In order to provide it with authentication information, open the following")
	fmt.Printf("URL, confirm that you allow %s to access your Twitter account and enter\nthe displayed PIN code.\n", PROGRAM_NAME)
	fmt.Printf("\nPlease open the following URL: %s\n", auth_url)
	fmt.Print("\nEnter PIN code: ")
	fmt.Scanln(&pin)

	tapi.SetPIN(pin)

	user, err := tapi.VerifyCredentials()
	if err != nil {
		return err
	}

	if err := SaveAccessToken(tapi.GetAccessToken(), cfgdir, *user.Screen_name); err != nil {
		return err
	}

	return nil
}
