package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	API_AVAILABILITY string = "http://archive.org/wayback/available?url="
	API_SAVE         string = "https://web.archive.org/save/"

	TIMEOUT time.Duration = 25
	client  *http.Client  = &http.Client{Timeout: TIMEOUT * time.Second}

	blacklist = []string{"cdn.discordapp.com", "discord.com", "tenor.com",
		"c.tenor.com", "archive.org", "web.archive.org", "youtu.be",
		"youtube.com", "www.youtube.com", "discord.gg", "media.discordapp.net"}
)

type Wayback struct {
	Snapshots Snapshot `json:"archived_snapshots,omitempty"`
}

type Snapshot struct {
	Recent Closest `json:"closest"`
}

type Closest struct {
	Available bool   `json:"available"`
	Status    string `json:"status"`
}

func isBlacklisted(host string) bool {

	for _, h := range blacklist {

		if host == h {
			return true
		}
	}
	return false
}

func isArchived(url string) (bool, int) {

	req, err := http.NewRequest("GET", API_AVAILABILITY+url, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return false, 0
	}
	av := &Wayback{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(av); err != nil {
		log.Println(err)
		return false, 0
	}
	status, _ := strconv.Atoi(av.Snapshots.Recent.Status)
	return av.Snapshots.Recent.Available, status
}

func archive(url string) int {

	req, err := http.NewRequest("GET", API_SAVE+url, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return 0
	}
	return resp.StatusCode
}
