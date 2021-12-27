package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestDB(t *testing.T) {

	// initDB()
	tmpDB, _ := ioutil.TempFile("", "tmp-*.db")
	db_path := tmpDB.Name()
	os.Remove(db_path)
	db := &SqliteDB{db: initDB(db_path)}

	// Cleanup temporary DB when test completes
	t.Cleanup(func() {
		os.Remove(db_path)
	})

	// AddArchived()
	m := Message{
		URL:     "http://example.com/",
		Author:  "000000000000000000",
		Guild:   "222222222222222222",
		Channel: "111111111111111111",
	}
	db.AddArchived(&m, 200)
	m = Message{
		URL:     "http://example.net/",
		Author:  "111111111111111111",
		Guild:   "222222222222222222",
		Channel: "333333333333333333",
	}
	db.AddArchived(&m, 404)

	// IsCached()
	url := "http://example.com/"
	cached, status_code := db.IsCached(url)
	if status_code != http.StatusOK || cached != true {
		t.Errorf("IsCached(): Received %t, %d; wanted %t, %d", cached,
			status_code, true, http.StatusOK)
	}
	url = "http://example.org/"
	cached, status_code = db.IsCached(url)
	if status_code != 0 || cached != false {
		t.Errorf("IsCached(): Received %t, %d; wanted %t, %d", cached,
			status_code, true, http.StatusOK)
	}

	// ListEntries()
	e, err := db.ListEntries(10, 0, "", "", "", "")
	if err != nil {
		t.Error(err)
	}
	if len(*e) != 2 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "000000000000000000", "", "", "")
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "", "222222222222222222", "", "")
	if len(*e) != 2 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "", "", "333333333333333333", "")
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "111111111111111111", "222222222222222222", "", "")
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "111111111111111111", "", "333333333333333333", "")
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "111111111111111111", "222222222222222222", "333333333333333333", "")
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "", "", "", "example")
	if len(*e) != 2 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}
	e, err = db.ListEntries(10, 0, "", "", "333333333333333333", "example")
	if len(*e) != 1 {
		t.Errorf("ListEntries(): Recieved length %d; wanted %d", len(*e), 2)
	}

	// Stats()
	stats, err := db.Stats()
	if err != nil {
		t.Fatal(err)
	}
	statsExpected := &Stats{
		URLs:     2,
		Users:    2,
		Guilds:   1,
		Channels: 2,
	}
	if stats == statsExpected {
		t.Errorf("Stats(): Received %v; wanted %v", stats, statsExpected)
	}
}
