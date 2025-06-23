package issues

type IssueComment struct {
	Action string `json:"action"` // "created", "edited", "deleted"

	Issue struct {
		Number      int                    `json:"number"`
		PullRequest map[string]interface{} `json:"pull_request"` // nil if not PR
	} `json:"issue"`

	Comment struct {
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`

	Repository struct {
		FullName string `json:"full_name"` // "owner/repo"
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`

	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}
