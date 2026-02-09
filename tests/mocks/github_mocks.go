package mocks

import (
	"ActQABot/conf"
	"ActQABot/pkg/github/gh_api"
	"github.com/google/go-github/v60/github"
	"testing"
)

var testToken = "test-token"

func mockGithub() {
	gh_api.Authorize = func(ghEnv conf.GithubAPIEnvironment, owner, repo string) (*github.InstallationToken, error) {
		return &github.InstallationToken{Token: &testToken}, nil
	}
}

func PostIssueCommentFixture(t *testing.T) chan *gh_api.BotResponse {
	mockGithub()
	original := gh_api.PostIssueCommentFunc
	call := make(chan *gh_api.BotResponse, 1)
	gh_api.PostIssueCommentFunc = func(botComment *gh_api.BotResponse, token string) error {
		call <- botComment
		if botComment.Text == "" {
			t.Errorf("expected botComment.Text to be non-empty")
		}
		if token != testToken {
			t.Errorf("expected token to be %s, got %s", testToken, token)
		}
		return nil
	}
	t.Cleanup(
		func() {
			gh_api.PostIssueCommentFunc = original
		},
	)
	return call
}
