package main

import (
	"log"

	"github.com/icanhazbroccoli/proglog/internal/server"
)

func main() {
	srv := server.NewHTTPServer("0.0.0.0:9090")
	log.Fatal(srv.ListenAndServe())
}
