package github_api

import "ActQABot/pkg/github/issues"

// IssueCommentEvent represents GitHub issue comment payload.
// @Description GitHub issue comment wrapper
type IssueCommentEvent struct {
	issues.IssueComment
}

// WebhookQuery represents optional query string.
// @Description options
type WebhookQuery struct {
	// If true, server will respond back to GitHub after processing
	PostBack bool `schema:"post_back" json:"post_back" example:"false"`
}

// LogStreamQuery defines query parameters for the log streaming endpoint.
// @Description Query parameters used to stream job logs.
type LogStreamQuery struct {
	// Hostname defined in configuration
	Host string `schema:"host" json:"host" example:"agent-01"`
	// Job ID whose logs will be streamed
	JobId string `schema:"job_id" json:"job_id" example:"job-abc-123"`
}

// HelpCommandResponse md text
// @Description
type HelpCommandResponse struct {
	Body string `json:"body"`
}

// CancelWorkflowQuery cancels workflow
// @Description cancels workflow
type CancelWorkflowQuery struct {
	// Job executor
	Host string `schema:"host" json:"host" example:"agent-01"`
	// Job to cancel
	JobId string `schema:"job_id" json:"job_id" example:"job-abc-123"`
}
