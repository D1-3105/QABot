package worker_api

import "github.com/gorilla/mux"

func Router() *mux.Router {
	r := mux.NewRouter().StrictSlash(false)
	r.HandleFunc("/worker/report/", reportCreate).Methods("POST")
	return r
}
