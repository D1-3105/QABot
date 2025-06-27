package main

import (
	"ActQABot/api/github_api"
	"ActQABot/api/static"
	"ActQABot/conf"
	_ "ActQABot/docs"
	"ActQABot/pkg/hosts"
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
	"strings"
)

// @title BeepBoop bot
// @version 1.0
// @description API for convenient CI/CD management
// @BasePath /api/v1

func mount(r *mux.Router, path string, handler http.Handler) {
	r.PathPrefix(path).Handler(
		http.StripPrefix(
			strings.TrimSuffix(path, "/"),
			handler,
		),
	)
}

func enableCORS(router *mux.Router) {
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

func main() {
	var err error
	flag.Parse()
	//
	conf.NewEnviron(&conf.GeneralEnvironments)
	conf.Hosts, err = conf.NewHostsEnvironment(conf.GeneralEnvironments.HostConf)
	if err != nil {
		panic(err)
	}
	hosts.HostAvbl = hosts.NewAvailability(conf.Hosts)
	conf.NewEnviron(&conf.GithubEnvironment)

	//server
	r := mux.NewRouter()
	enableCORS(r)
	mount(r, "/api/v1", github_api.Router())
	mount(r, "/static", static.Router())
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	var serverEnv conf.ServerEnvironment
	conf.NewEnviron(&serverEnv)
	glog.Infof("Listening on %s", serverEnv.Address)
	err = http.ListenAndServe(serverEnv.Address, r)
	if err != nil {
		glog.Error("Error starting server:", err)
	}
}
