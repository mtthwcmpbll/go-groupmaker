package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()

	// Set up the basic web services
	router.GET("/status", Status)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK!")
}
