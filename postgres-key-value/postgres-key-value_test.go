package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"fmt"
	"github.com/bkeroack/travel"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func setupDB() {
	var err error
	db, err = sql.Open("postgres", "postgres://postgres:postgres@localhost/keyvalue_testing?sslmode=disable")
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS root_tree;")
	if err != nil {
		log.Fatalf("Error dropping root_tree: %v\n", err)
	}
	_, err = db.Exec(createDb)
	if err != nil {
		log.Fatalf("Error creating root_tree: %v\n", err)
	}
	_, err = db.Exec(initialRootTree)
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

func random_range(max int64) int64 {
	max_big := *big.NewInt(max)
	n, err := rand.Int(rand.Reader, &max_big)
	if err != nil {
		log.Fatalf("ERROR: cannot get random integer!\n")
	}
	return n.Int64()
}

func random_string(l uint) string {
	chars := []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	output := make([]byte, l)
	for i := 0; i < len(output); i++ {
		output[i] = chars[random_range(int64(len(chars)))]
	}
	return string(output)
}

var wg sync.WaitGroup

func randomInsert(url string, k string, v string) {
	defer wg.Done()
	wt := random_range(int64(50))
	time.Sleep(time.Duration(wt) * time.Millisecond)
	req, err := http.NewRequest("PUT", fmt.Sprintf("%v/%v", url, k), bytes.NewBuffer([]byte(v)))
	if err != nil {
		log.Fatalf("Creating request object failed: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		log.Fatalf("Request failed: k: %v, v: %v: %v\n", k, v, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: k: %v, v: %v: %v\n", k, v, err)
	}
	log.Printf("RESPONSE: Status: %v ; Body: %v\n", resp.Status, string(body))
}

func TestSimultaneousUpdates(t *testing.T) {
	cc := 100
	setupDB()
	r := createRouter()
	s := httptest.NewServer(r)
	inserts := make(map[string]string, cc)
	for i := 0; i <= cc; i++ {
		k := random_string(uint(16))
		v := random_string(uint(16))
		inserts[k] = v
		wg.Add(1)
		go randomInsert(s.URL, k, v)
	}
	log.Printf("All requests spawned, waiting")
	wg.Wait()
}
