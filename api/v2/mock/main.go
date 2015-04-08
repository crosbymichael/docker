package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/docker/docker/api/v2"
)

var swaggerPath string

func init() {
	flag.StringVar(&swaggerPath, "swagger", "", "path to swagger dist files")
	flag.Parse()
}

func main() {
	if err := http.ListenAndServe("127.0.0.1:5000", v2.New(nil, swaggerPath)); err != nil {
		log.Fatal(err)
	}
}
