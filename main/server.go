package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func index(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	log.Println(string(body), err)
	b := bytes.Repeat([]byte("hello"), 100000)
	w.Write(b)
}

func main() {
	http.HandleFunc("/", index)
	log.Println(http.ListenAndServe(":8000", nil))
}
