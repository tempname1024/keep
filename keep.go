package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/publicsuffix"
)

type Config struct {
	Token   string   `json:"token"`
	Verbose bool     `json:"verbose"`
	Ignore  []string `json:"ignore"`
}

type Message struct {
	URL     string
	Author  string
	Guild   string
	Channel string
}

var (
	messageChan chan *Message
	config      Config
)

func main() {

	// ~/.keep directory stores db cache and json config
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	keepDir := path.Join(user.HomeDir, ".keep")

	// Default config location: ~/.keep/keep.json
	var configPath string
	flag.StringVar(&configPath, "config", path.Join(keepDir, "keep.json"),
		"path to configuration file")
	flag.Parse()
	conf, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal([]byte(conf), &config)
	if err != nil {
		log.Fatal(err)
	}

	// Create and initialize URL cache database
	db := initDB(path.Join(keepDir, "keep.db"))

	// Channel for passing URLs to the archive goroutine for archival
	messageChan = make(chan *Message, 25)
	go archiver(db)

	// Create a new Discord session using provided credentials
	dg, err := discordgo.New(config.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Make our client look like Firefox since we're authenticating with
	// user/pass credentials (self bot)
	dg.UserAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:94.0) Gecko/20100101 Firefox/94.0 "

	// Register the messageCreate func as a callback for MessageCreate events
	dg.AddHandler(messageCreate)

	// We only care about receiving message events
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session
	dg.Close()
}

// archiver is intended to be run in its own goroutine, receiving URLs from main
// over a shared channel for processing
func archiver(db *sql.DB) {

	// Each iteration removes and processes one url from the channel
	for {

		// Blocks until URL is received
		message := <-messageChan

		// Skip if we have URL in database
		cached, _ := isCached(db, message.URL)
		if cached {
			continue
		}

		// Skip if the Internet Archive already has a copy available
		archived, status_code := isArchived(message.URL)
		if archived && status_code == http.StatusOK {
			addArchived(db, message, status_code)
			log.Printf("SKIP %d %s", status_code, message.URL)
			continue
		}

		// Archive, URL is not present in cache or IA
		status_code = archive(message.URL)
		addArchived(db, message, status_code)
		log.Printf("SAVE %d %s", status_code, message.URL)

		// Limit requests to Wayback API to 5-second intervals
		time.Sleep(5 * time.Second)
	}
}

// messageCreate be called (due to AddHandler above) every time a new message is
// created on any channel that the authenticated bot has access to
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// https://github.com/bwmarrin/discordgo/issues/961
	if m.Content == "" {
		chanMsgs, err := s.ChannelMessages(m.ChannelID, 1, "", "", m.ID)
		if err != nil {
			log.Printf("Unable to get messages: %s", err)
			return
		}
		if len(chanMsgs) > 0 {
			m.Content = chanMsgs[0].Content
			m.Attachments = chanMsgs[0].Attachments
		}
	}

	// Log all messages if verbose set to true
	if config.Verbose {
		log.Println(m.Content)
	}

	// Split message by spaces into individual fields
	for _, w := range strings.Fields(m.Content) {

		// Assess whether message part looks like a valid URL
		u, err := url.Parse(w)
		if err != nil || !u.IsAbs() || strings.IndexByte(u.Host, '.') <= 0 {
			continue
		}

		// Ensure domain TLD is ICANN-managed
		if _, icann := publicsuffix.PublicSuffix(u.Host); !icann {
			continue
		}

		// Ensure host is not present in ignoreList set
		if isIgnored(config.Ignore, w) {
			continue
		}

		// Send message attributes/URL over the channel
		message := Message{
			URL:     w,
			Author:  m.Author.ID,
			Guild:   m.GuildID,
			Channel: m.ChannelID,
		}
		messageChan <- &message
	}
}
