package github_api

import "ActQABot/pkg/github/issues"

type IssueCommentEvent struct {
	issues.IssueComment
}

type LogStreamQuery struct {
	Host  string `schema:"host"`
	JobId string `schema:"job_id"`
}
