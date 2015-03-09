package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bkeroack/travel"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	//"os"
)

var db *sql.DB

func get_root_tree() (map[string]interface{}, error) {
	var tree []byte
	err := db.QueryRow("SELECT tree FROM root_tree order by id DESC LIMIT 1;").Scan(&tree) // order by sequential id
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("Error getting root tree: %v\n", err)
	}
	var rt map[string]interface{}
	err = json.Unmarshal(tree, &rt)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("Error deserializing root tree: %v\n", err)
	}
	return rt, nil
}

func save_root_tree(rt map[string]interface{}) error {
	b, err := json.Marshal(rt)
	if err != nil {
		return fmt.Errorf("Error serializing root tree: %v\n", err)
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("Error starting transaction: %v\n", err)
	}
	defer tx.Rollback()
	_, err = tx.Exec("INSERT INTO root_tree (tree) VALUES (?)", b)
	if err != nil {
		return fmt.Errorf("Error inserting root tree: %v\n", err)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Error committing root tree transaction: %v\n", err)
	}
	return nil
}

// This handler runs for every valid request
func PrimaryHandler(w http.ResponseWriter, r *http.Request, c *travel.Context) {
	save_rt := func() bool {
		_, err := db.Exec("LOCK TABLE root_tree IN ACCESS EXCLUSIVE MODE;")
		defer db.Exec("")
		if err != nil {
			log.Fatalf("Error locking root_tree table: %v\n", err)
		}
		err = c.Refresh()
		if err != nil {
			return false
		}
		err = save_root_tree(c.RootTree)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving root tree: %v", err), http.StatusInternalServerError)
		}
		return err == nil
	}

	json_output := func(val interface{}) {
		b, err := json.Marshal(val)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error serializing output: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}

	switch r.Method {
	case "GET":
		json_output(c.CurrentObj) // CurrentObj is the object returned after full traveral; eg '/foo/bar': CurrentObj = root_tree["foo"]["bar"]
	case "PUT":
		d := json.NewDecoder(r.Body)
		var b interface{}
		err := d.Decode(&b)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not serialize request body: %v", err), http.StatusBadRequest)
			return
		}
		k := c.Path[len(c.Path)-1]
		c.CurrentObj.(map[string]interface{})[k] = b //maps are reference types, so a modification to CurrentObj is reflected in RootTree
		if save_rt() {
			w.Header().Set("Location", fmt.Sprintf("http://%v/%v", r.Host, r.URL.Path))
			json_output(map[string]string{
				"success": "value written",
			})
		}
		http.Error(w, "Error saving value", http.StatusInternalServerError)
		return
	case "DELETE":
		po, err := c.WalkBack(1) // We need to get the object one node up in the root tree, so we can delete the current object
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		delete(po, c.Path[len(c.Path)-1]) // delete the node from the last path token, which must exist otherwise the req would have 404ed
		if save_rt() {
			json_output(map[string]string{
				"success": "value deleted",
			})
		}
		http.Error(w, "Error deleting value", http.StatusInternalServerError)
		return
	default:
		w.Header().Set("Accepts", "GET,PUT,DELETE")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Travel runs this in the event of error conditions (including 404s, etc)
func ErrorHandler(w http.ResponseWriter, r *http.Request, err travel.TraversalError) {
	http.Error(w, err.Error(), err.Code())
}

func init() {
	var err error
	db, err = sql.Open("postgres", "postgres://postgres:postgres@localhost/keyvalue?sslmode=disable")
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}
	setupTables()
}

func main() {
	defer db.Close()
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
	http.Handle("/", r)
	http.ListenAndServe("0.0.0.0:8000", nil)
}
