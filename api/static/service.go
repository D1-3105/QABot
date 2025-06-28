package static

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"net/http"
	"path/filepath"
)

func Router(staticDir string) *mux.Router {
	frontendRouter := mux.NewRouter()
	fs := http.FileServer(http.Dir(staticDir))

	frontendRouter.PathPrefix("").Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filePath := filepath.Join(staticDir, r.URL.Path)
			glog.V(1).Infof("Serving static file: %s\n", filePath)
			fs.ServeHTTP(w, r)
		}),
	)

	return frontendRouter
}
