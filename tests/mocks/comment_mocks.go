package mocks

type MockComment struct {
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

type IssueCommentPayload struct {
	Action       string      `json:"action"`
	IssueComment MockComment `json:"comment"`
}
