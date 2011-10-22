package main

import (
	"fmt"
	"os"
	"os/user"
	"json"
	"io/ioutil"
	"flag"
	"log"
	oauth "github.com/akrennmair/goauth"
	goconf "goconf.googlecode.com/hg"
)

const (
	PROGRAM_NAME = "gockel"
	PROGRAM_VERSION = "0.0"
	PROGRAM_URL = "https://github.com/akrennmair/gockel"
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

	tapi := NewTwitterAPI("sDggzGbHbyAfl5fJ87XOCA", "MOCQDL7ot7qIxMwYL5x1mMAqiYBYxNTxPWS6tc6hw", cfg)

	at, err := LoadAccessToken(cfgdir)

	if err == nil {
		tapi.SetAccessToken(at)
	} else {
		auth_url, err := tapi.GetRequestAuthorizationURL()
		if err != nil {
			log.Printf("GetRequestAuthorizationURL failed: %v", err)
			fmt.Fprintf(os.Stderr, "GetRequestAuthorizationURL failed: %v\n", err)
			return
		}

		var pin string
		fmt.Printf("%s doesn't yet have information how to access your Twitter account.\n", PROGRAM_NAME)
		fmt.Println("In order to provide it with authentication information, open the following")
		fmt.Printf("URL, confirm that you allow %s to access your Twitter account and enter\nthe displayed PIN code.\n", PROGRAM_NAME)
		fmt.Printf("\nPlease open the following URL: %s\n", auth_url)
		fmt.Print("\nEnter PIN code: ")
		fmt.Scanln(&pin)

		tapi.SetPIN(pin)

		if err := SaveAccessToken(tapi.GetAccessToken(), cfgdir); err != nil {
			fmt.Printf("saving access token failed: %v\n", err)
			return
		}
	}

	cmdchan := make(chan TwitterCommand, 1)
	newtweetchan := make(chan []*Tweet, 1)
	lookupchan := make(chan TweetRequest, 1)
	uiactionchan := make(chan UserInterfaceAction, 10)

	model := NewModel(tapi, cmdchan, newtweetchan, lookupchan, uiactionchan, cfg)
	go model.Run()

	ui := NewUserInterface(cmdchan, newtweetchan, lookupchan, uiactionchan, cfg)
	go ui.Run()

	ui.InputLoop()
}

func SaveAccessToken(at *oauth.AccessToken, cfgdir string) os.Error {
	data, marshalerr := json.Marshal(at)
	if marshalerr != nil {
		return marshalerr
	}

	f, ferr := os.OpenFile(cfgdir + "/access_token.json", os.O_WRONLY|os.O_CREATE, 0600)
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

func LoadAccessToken(cfgdir string) (*oauth.AccessToken, os.Error) {
	f, ferr := os.Open(cfgdir + "/access_token.json")
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
