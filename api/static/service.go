package static

import (
	"github.com/gorilla/mux"
	"net/http"
)

func Router() *mux.Router {
	frontendRouter := mux.NewRouter()
	staticDir := "./frontend/web-interface/dist"
	fs := http.FileServer(http.Dir(staticDir))
	frontendRouter.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	return frontendRouter
}
