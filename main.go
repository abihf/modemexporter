package main

import (
	"net/http"
	"os"
)

func main() {
	server := Server{
		Modem:    &Huawei{URL: os.Getenv("MODEM_URL")},
		User:     os.Getenv("MODEM_USER"),
		Password: os.Getenv("MODEM_PASSWORD"),
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, &server)
}
