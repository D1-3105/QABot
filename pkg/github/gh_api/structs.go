package gh_api

type BotResponse struct {
	Owner       string
	Repo        string
	IssueNumber int
	CommentId   *int

	Text string
}

type Comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}
