package tests

import (
	"ActQABot/api/worker_api"
	"ActQABot/pkg/worker_report"
	"ActQABot/tests/mocks"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_WorkerReportCreate(t *testing.T) {
	setupTestEnv(t)
	_ = mocks.MockWorkerReportEtcd(nil, nil)
	newWorkerReport := worker_report.JobReport{JobId: "123", JobReportText: "Something"}
	body, err := json.Marshal(newWorkerReport)
	if err != nil {
		t.Errorf("failed to marshal newWorkerReport %v", err)
	}
	req := httptest.NewRequest("POST", "/worker/report/", bytes.NewBuffer(body))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	w := httptest.NewRecorder()
	router := worker_api.Router()
	router.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("failed to read response body %v", err)
	}
	jobReports, err := worker_report.SubscribeJobReports(t.Context())
	if err != nil {
		t.Errorf("failed to subscribe job reports %v", err)
	}
	select {
	case report := <-jobReports:
		require.Condition(
			t, func() bool {
				return report.Report.JobId == newWorkerReport.JobId && report.Report.JobReportText == newWorkerReport.
					JobReportText
			}, "Reports are not equal!",
		)
		break
	case <-time.After(30 * time.Second):
		t.Errorf("timed out waiting for job report")
	}
}
