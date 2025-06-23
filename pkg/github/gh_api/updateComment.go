package gh_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func UpdateIssueComment(botComment BotResponse, token string) error {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/issues/comments/%d",
		botComment.Owner,
		botComment.Repo,
		botComment.CommentId,
	)

	payload := map[string]string{
		"body": botComment.Text,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
