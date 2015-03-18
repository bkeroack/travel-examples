package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bkeroack/travel"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

const RTLock = iota

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
	_, err = tx.Exec("INSERT INTO root_tree (tree) VALUES ($1)", b)
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

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening transaction: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	lock_and_refresh := func() travel.TraversalError {
		_, err := tx.Exec("SELECT pg_advisory_lock($1);", RTLock)
		if err != nil {
			log.Printf("Error locking root_tree: %v\n", err)
			return travel.InternalError(err.Error())
		}
		return c.Refresh()
	}

	unlock := func() {
		_, err := tx.Exec("SELECT pg_advisory_unlock($1);", RTLock)
		err = tx.Commit()
		if err != nil {
			log.Printf("Error unlocking root_tree: %v\n", err)
		}
	}

	save_rt := func() bool {
		err := save_root_tree(c.RootTree)
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

	check_empty := func() bool {
		if len(c.Path) == 1 && c.Path[0] == "" {
			http.Error(w, "Bad Request: key is required", http.StatusBadRequest)
			return false
		}
		return true
	}

	switch r.Method {
	case "GET":
		json_output(c.CurrentObj) // CurrentObj is the object returned after full traveral; eg '/foo/bar': CurrentObj = root_tree["foo"]["bar"]
	case "PUT":
		if !check_empty() {
			return
		}
		d := json.NewDecoder(r.Body)
		var b interface{}
		err := d.Decode(&b)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not serialize request body: %v", err), http.StatusBadRequest)
			return
		}
		terr := lock_and_refresh()
		defer unlock()
		if terr != nil {
			http.Error(w, terr.Error(), terr.Code())
			return
		}
		var co map[string]interface{}
		if len(c.Subpath) == 0 { // key exists
			po, wberr := c.WalkBack(1) // c.CurrentObj is the *value*, so we have to walk back one
			if wberr != nil {
				http.Error(w, terr.Error(), http.StatusInternalServerError)
			}
			co = po
		} else { // key doesn't exist yet
			co = c.CurrentObj.(map[string]interface{})
		}
		k := c.Path[len(c.Path)-1]
		co[k] = b //maps are reference types, so a modification to CurrentObj is reflected in RootTree
		if save_rt() {
			log.Printf("Write: key: %v ; value: %v\n", k, b)
			w.Header().Set("Location", fmt.Sprintf("http://%v/%v", r.Host, r.URL.Path))
			json_output(map[string]string{
				"success": "value written",
			})
			return
		}
		http.Error(w, "Error saving value", http.StatusInternalServerError)
		return
	case "DELETE":
		if !check_empty() {
			return
		}
		err := lock_and_refresh()
		defer unlock()
		if err != nil {
			http.Error(w, err.Error(), err.Code())
			return
		}
		po, terr := c.WalkBack(1) // We need to get the object one node up in the root tree, so we can delete the current object
		if terr != nil {
			http.Error(w, terr.Error(), http.StatusInternalServerError)
			return
		}
		k := c.Path[len(c.Path)-1]
		delete(po, k) // delete the node from the last path token, which must exist otherwise the req would have 404ed
		if save_rt() {
			log.Printf("Delete: key: %v\n", k)
			json_output(map[string]string{
				"success": "value deleted",
			})
			return
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
	log.Printf("Listening on port 8000")
	http.ListenAndServe("0.0.0.0:8000", nil)
}
