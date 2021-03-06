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
	"keep/normalize"
)

type Config struct {
	Token   string   `json:"token"`
	Verbose bool     `json:"verbose"`
	Ignore  []string `json:"ignore"`
	Host    string   `json:"host"`
	Port    string   `json:"port"`
}

type Message struct {
	URL     string
	Author  string
	Guild   string
	Channel string
}

type SqliteDB struct {
	db *sql.DB
}

var (
	messageChan chan *Message
	config      Config
)

func main() {

	// Directory (default ~/.keep) containing configuration and DB cache
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	var keepDir string
	flag.StringVar(&keepDir, "path", path.Join(user.HomeDir, ".keep"),
		"path to data directory")
	flag.Parse()

	// See ./keep.json for set of supported parameters/values
	configPath := path.Join(keepDir, "keep.json")
	conf, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal([]byte(conf), &config)
	if err != nil {
		log.Fatal(err)
	}

	// Create and initialize URL cache database
	sqlSqliteDB := initDB(path.Join(keepDir, "keep.db"))
	db := &SqliteDB{db: sqlSqliteDB}

	// Channel for passing URLs to the archive goroutine for archival
	messageChan = make(chan *Message, 25)
	go archiver(db)

	// Start HTTP server
	http.HandleFunc("/", db.IndexHandler)
	log.Printf("Listening on %v port %v (http://%v:%v/)\n", config.Host,
		config.Port, config.Host, config.Port)
	go http.ListenAndServe(fmt.Sprintf("%s:%s", config.Host, config.Port), nil)

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
func archiver(db *SqliteDB) {

	// Each iteration removes and processes one url from the channel
	for {

		// Blocks until URL is received
		message := <-messageChan

		// Skip if we've already seen URL (cached)
		cached, status_code := db.IsCached(message.URL)
		if cached {
			log.Println("SEEN", status_code, message.URL)
			continue
		}

		// Skip if the Internet Archive already has a copy available
		archived, status_code := isArchived(message.URL)
		if archived && status_code == http.StatusOK {
			db.AddArchived(message, status_code)
			log.Println("SKIP", status_code, message.URL)
			continue
		}

		// Archive, URL is not present in cache or IA
		status_code = archive(message.URL)
		db.AddArchived(message, status_code)
		log.Println("SAVE", status_code, message.URL)

		// Limit requests to Wayback API to 15-second intervals
		time.Sleep(15 * time.Second)
	}
}

// messageCreate be called (due to AddHandler above) every time a new message is
// created on any channel that the authenticated bot has access to
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// https://github.com/bwmarrin/discordgo/issues/961
	if m.Content == "" {
		chanMsgs, err := s.ChannelMessages(m.ChannelID, 1, "", "", m.ID)
		if err != nil {
			log.Println("Unable to get messages:", err)
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

		// Normalize URL (RFC 3986)
		uStr := normalize.NormalizeURL(u,
			normalize.FlagsSafe|normalize.FlagRemoveDotSegments|
				normalize.FlagRemoveDuplicateSlashes|
				normalize.FlagRemoveFragment|
				normalize.FlagSortQuery)

		// Ensure host is not present in ignoreList set
		if isIgnored(config.Ignore, uStr) {
			continue
		}

		// Send message attributes/URL over the channel
		message := Message{
			URL:     uStr,
			Author:  m.Author.ID,
			Guild:   m.GuildID,
			Channel: m.ChannelID,
		}
		messageChan <- &message
	}
}
