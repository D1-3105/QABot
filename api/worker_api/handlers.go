package worker_api

import (
	"ActQABot/api/base_api"
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"net/http"
)

func reportCreate(w http.ResponseWriter, r *http.Request) {
	var report *JobWorkerReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		base_api.APIReturnError(w, err)
		glog.Errorf("report validation error (decoding): %v", err)
		return
	}

	if err := report.SendEvent(context.Background()); err != nil {
		base_api.APIReturnError(w, err)
		glog.Errorf("report validation error (SendEvent): %v", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(JobReportResponse{}); err != nil {
		glog.Errorf("report validation error (Encode result): %v", err)
		return
	}
	return
}
