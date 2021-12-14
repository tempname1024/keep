package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var (
	API_AVAILABILITY string = "http://archive.org/wayback/available?url="
	API_SAVE         string = "https://web.archive.org/save/"

	TIMEOUT time.Duration = 10
	client  *http.Client  = &http.Client{Timeout: TIMEOUT * time.Second}
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

func isIgnored(regex []string, url string) bool {

	for _, r := range regex {

		if v := regexp.MustCompile(r); v.MatchString(url) {
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
		if e, _ := err.(net.Error); !e.Timeout() {
			log.Println(err)
		}
		return 0
	}
	return resp.StatusCode
}
