package main

import (
	"net/http"
	"os"

	"github.com/abihf/modemexporter/modem"
	_ "github.com/abihf/modemexporter/modem/huawei/eg8141A5"
)

func main() {
	m, err := modem.FromEnv()
	if err != nil {
		panic(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := modem.Server{Modem: m}
	err = http.ListenAndServe(":"+port, &server)
	if err != nil {
		panic(err)
	}
}
