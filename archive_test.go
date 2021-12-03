package main

import (
	"net/http"
	"testing"
)

func TestIsArchived(t *testing.T) {

	url := "http://example.com/"
	archived, status := isArchived(url)
	if archived != true || status != 200 {
		t.Errorf("Received %t, %d: want %t, %d", archived, status, true, 200)
	}
}

func TestArchive200(t *testing.T) {

	url := "http://example.com/"
	status := archive(url)
	if status != http.StatusOK {
		t.Errorf("Recieved %d; want %d", status, http.StatusOK)
	}
}
