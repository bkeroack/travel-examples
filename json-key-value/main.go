package main

import (
	"encoding/json"
	"fmt"
	"github.com/bkeroack/travel"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	root_tree_path = "root_tree.json"
)

func get_root_tree() (map[string]interface{}, error) {
	var v map[string]interface{}
	d, err := ioutil.ReadFile(root_tree_path)
	if err != nil {
		return map[string]interface{}{}, err
	}
	json.Unmarshal(d, v)
	return v, nil
}

func PrimaryHandler(w http.ResponseWriter, r *http.Request, c *travel.Context) {
	log.Printf("PrimaryHandler: %v\n", c)

	val, err := json.Marshal(c.CurrentObj)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't marshal context: %v", err), 500)
		return
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		w.Write(val)
	case "PUT":
		d := json.NewDecoder(r.Body)
		var b interface{}
		err := d.Decode(&b)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not serialize request body: %v", err), 500)
			return
		}
	}
}

func ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("ErrorHandler: %v\n", err)
	http.Error(w, err.Error(), err.Code)
}

func main() {
	hm := map[string]travel.TravelHandler{
		"": PrimaryHandler,
	}
	options := travel.TravelOptions{
		SubpathMaxLength: map[string]int{
			"GET":    0,
			"PUT":    1,
			"DELETE": 0,
		},
	}
	r := travel.NewRouter(get_root_tree, hm, ErrorHandler, &options)
	http.Handle("/", r)
	http.ListenAndServe("127.0.0.1:8000", nil)
}
