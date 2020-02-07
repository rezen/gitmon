package main

import (
	"net/http"
	"log"
	"github.com/rezen/gitmon"
	_ "net/http/pprof"
)

func main() {
	go func() {
		// @todo middleware to ensure only local addresses
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	gitmon.Worker()
	
}