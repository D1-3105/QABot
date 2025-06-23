package gh_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func getBotComments(token, owner, repo string, issueNumber int, botLogin string) ([]Comment, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", owner, repo, issueNumber)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var allComments []Comment
	if err := json.NewDecoder(resp.Body).Decode(&allComments); err != nil {
		return nil, err
	}

	var botComments []Comment
	for _, c := range allComments {
		if c.User.Login == botLogin {
			botComments = append(botComments, c)
		}
	}

	return botComments, nil
}
