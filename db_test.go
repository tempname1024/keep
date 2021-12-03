package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

var (
	db      *sql.DB
	db_path string
)

func TestInitDB(t *testing.T) {

	tmpDB, _ := ioutil.TempFile("", "tmp-*.db")
	db_path = tmpDB.Name()
	os.Remove(db_path)
	db = initDB(db_path)
}

func TestAddArchived(t *testing.T) {

	m := Message{
		URL:     "http://example.com/",
		Author:  "000000000000000000",
		Guild:   "000000000000000000",
		Channel: "000000000000000000",
	}
	addArchived(db, &m, 200)
}

func TestIsCached(t *testing.T) {

	url := "http://example.com/"
	cached, status_code := isCached(db, url)
	if status_code != http.StatusOK || cached != true {
		t.Errorf("Received %t, %d; wanted %t, %d", cached, status_code, true,
			http.StatusOK)
	}
}

func TestDBCleanup(t *testing.T) {

	os.Remove(db_path)
}
