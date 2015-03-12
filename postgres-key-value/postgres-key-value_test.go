package main

import (
	"database/sql"
	"github.com/bkeroack/travel"
	_ "github.com/lib/pq"
	"net/http/httptest"
	"testing"
)

var db *sql.DB

func setupDB() {
	var err error
	db, err = sql.Open("postgres", "postgres://postgres:postgres@localhost/keyvalue_testing?sslmode=disable")
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}
	err = db.Exec("DROP TABLE IF EXISTS root_tree;")
	if err != nil {
		log.Fatalf("Error dropping root_tree: %v\n", err)
	}
	err = db.Exec(createDb)
	if err != nil {
		log.Fatalf("Error creating root_tree: %v\n", err)
	}
	err = db.Exec(initialRootTree)
	if err != nil {
		log.Fatalf("Error inserting initial root_tree: %v\n", err)
	}
}

func createRouter() *travel.Router {
	hm := map[string]travel.TravelHandler{
		"": PrimaryHandler,
	}
	options := travel.TravelOptions{
		StrictTraversal:   true,
		UseDefaultHandler: true, // DefaultHandler is empty string by default (zero value for string)
		SubpathMaxLength: map[string]int{
			"GET":    0,
			"PUT":    1,
			"DELETE": 0,
		},
	}
	r, err := travel.NewRouter(get_root_tree, hm, ErrorHandler, &options)
	if err != nil {
		log.Fatalf("Error creating Travel router: %v\n", err)
	}
	return r
}

func TestSimultaneousUpdates(t *testing.T) {
	setupDB()
	r := createRouter()
	s := httptest.NewServer(r)
}
