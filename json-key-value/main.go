package main

import (
	"encoding/json"
	"fmt"
	"github.com/bkeroack/travel"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	err = json.Unmarshal(d, &v)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return v, nil
}

func save_root_tree(rt map[string]interface{}) error {
	b, err := json.Marshal(rt)
	if err != nil {
		return err
	}
	f, err := os.Create(root_tree_path)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

// This handler runs for every valid request
func PrimaryHandler(w http.ResponseWriter, r *http.Request, c *travel.Context) {
	save_rt := func() bool {
		err := save_root_tree(c.RootTree)
		if err != nil {
			http.Error(w, fmt.Sprintf("error saving root tree: %v", err), http.StatusInternalServerError)
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
			http.Error(w, fmt.Sprintf("could not serialize request body: %v", err), http.StatusBadRequest)
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

func main() {
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
	http.ListenAndServe("127.0.0.1:8000", nil)
}
