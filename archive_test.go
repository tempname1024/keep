package main

import (
	"net/http"
	"testing"
)

func TestIsArchived(t *testing.T) {

	url := "http://example.com/"
	archived, status := isArchived(url)
	if !archived || status != 200 {
		t.Errorf("Received %t, %d; want %t, %d", archived, status, true, 200)
	}
}

func TestIsNotArchived(t *testing.T) {

	url := "http://invalidurl.local/"
	archived, _ := isArchived(url)
	if archived {
		t.Errorf("Received %t; want %t", archived, false)
	}
}

func TestIsIgnored(t *testing.T) {

	ignoreRegex := []string{`^https?://([^/]*\.)?example\.[^/]+/`}
	url := "https://example.com/path"
	ignored := isIgnored(ignoreRegex, url)
	if !ignored {
		t.Errorf("Received %t; want %t", ignored, true)
	}
}

func TestIsNotIgnored(t *testing.T) {

	ignoreRegex := []string{`^https?://([^/]*\.)?example\.[^/]+/`}
	url := "https://google.com/path"
	ignored := isIgnored(ignoreRegex, url)
	if ignored {
		t.Errorf("Received %t; want %t", ignored, false)
	}
}

func TestArchive200(t *testing.T) {

	url := "http://example.com/"
	status := archive(url)
	if status != http.StatusOK {
		t.Errorf("Recieved %d; want %d", status, http.StatusOK)
	}
}
