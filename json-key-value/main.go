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

func json_output(w http.ResponseWriter, val interface{}) {
	b, err := json.Marshal(val)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error serializing output: %v", err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func PrimaryHandler(w http.ResponseWriter, r *http.Request, c *travel.Context) {
	log.Printf("PrimaryHandler: %v\n", c)

	save_rt := func() bool {
		err := save_root_tree(c.RootTree)
		if err != nil {
			http.Error(w, fmt.Sprintf("error saving root tree: %v", err), 500)
		}
		return err == nil
	}

	switch r.Method {
	case "GET":
		json_output(w, c.CurrentObj)
	case "PUT":
		d := json.NewDecoder(r.Body)
		var b interface{}
		err := d.Decode(&b)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not serialize request body: %v", err), 500)
			return
		}
		k := c.Subpath[0]
		c.CurrentObj.(map[string]interface{})[k] = b
		if save_rt() {
			json_output(w, map[string]string{
				"success": "value written",
			})
		}
		return
	case "DELETE":
		po := c.WalkBack(1)
		delete(po, c.Path[len(c.Path)-1])
		if save_rt() {
			json_output(w, map[string]string{
				"success": "value deleted",
			})
		}
		return
	default:
		w.Header().Set("Accepts", "GET,PUT,DELETE")
		http.Error(w, "Method Not Allowed", 405)
	}
}

func ErrorHandler(w http.ResponseWriter, r *http.Request, err travel.TraversalError) {
	log.Printf("ErrorHandler: %v\n", err)
	http.Error(w, err.Error(), err.Code())
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
