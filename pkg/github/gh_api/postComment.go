package gh_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io"
	"net/http"
)

var PostIssueCommentFunc = postIssueComment

func postIssueComment(botComment *BotResponse, token string) error {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/issues/%d/comments",
		botComment.Owner,
		botComment.Repo,
		botComment.IssueNumber,
	)

	payload := map[string]string{
		"body": botComment.Text,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		glog.Errorf("Error marshalling payload: %v", err)
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		glog.Errorf("Error creating request: %v", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("post issue comment failed with error: %v", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 201 {
		glog.Errorf("Post comment failed with status code %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
