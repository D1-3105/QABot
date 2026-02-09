package worker_api

import (
	"ActQABot/api/base_api"
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"net/http"
)

// reportCreate handles the creation of a job worker report.
// @Summary Create a worker report
// @Description Decodes worker report data, sends a business event, and returns a success response.
// @Tags reports
// @Accept json
// @Produce json
// @Param report body JobWorkerReport true "Job Worker Report Data"
// @Success 201 {object} JobReportResponse "Report successfully created"
// @Failure 400 {object} base_api.APIError "Invalid JSON or validation error"
// @Failure 500 {object} base_api.APIError "Internal server error during event processing"
// @Router /worker/report/ [post]
func reportCreate(w http.ResponseWriter, r *http.Request) {
	var report *JobWorkerReport

	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		base_api.APIReturnError(w, err)
		glog.Errorf("report validation error (decoding): %v", err)
		return
	}
	report.Retried = new(int32)
	*report.Retried = 0

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
