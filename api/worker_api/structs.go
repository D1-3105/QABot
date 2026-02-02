package worker_api

type JobWorkerReport struct {
	JobId      string `json:"job_id"`
	ReportText string `json:"report_text"`
}

type JobReportStored struct {
}
