package github_api

import "github.com/gorilla/mux"

func Router() *mux.Router {
	r := mux.NewRouter().StrictSlash(false)
	r.HandleFunc("/github/events/", webhookHandler).Methods("POST")
	r.HandleFunc("/job/logs/", logStreamer).Methods("GET", "OPTIONS")
	r.HandleFunc("/help", helpCommand).Methods("GET", "OPTIONS")
	return r
}
