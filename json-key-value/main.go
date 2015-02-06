package main

import (
	"encoding/json"
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
}

func ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("ErrorHandler: %v\n", err)
}

func main() {
	hm := map[string]travel.TravelHandler{
		"": PrimaryHandler,
	}
	r := travel.NewRouter(get_root_tree, hm, ErrorHandler, nil)
	http.Handle("/", r)
	http.ListenAndServe("127.0.0.1:8000", nil)
}
