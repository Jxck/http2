package main

import (
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
	w.Write(body)
}

func main() {
	http.HandleFunc("/", index)
	log.Println(http.ListenAndServe(":8000", nil))
}
