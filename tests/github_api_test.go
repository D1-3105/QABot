package tests

import (
	"ActQABot/conf"
	"ActQABot/pkg/github/gh_api"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthForRepo(t *testing.T) {
	const (
		testOwner       = "testowner"
		testRepo        = "testrepo"
		expectedInstall = int64(123456)
		expectedToken   = "mocked_installation_token"
	)
	tmpFile := t.TempDir() + "/mock.key"
	generateRSAPrivateKeyPEM(t, tmpFile, 2048)
	mux := http.NewServeMux()

	mux.HandleFunc(
		fmt.Sprintf("/api/v3/repos/%s/%s/installation", testOwner, testRepo),
		func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Bearer test-jwt", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": expectedInstall,
			})
		},
	)
	mux.HandleFunc(
		fmt.Sprintf("/api/v3/app/installations/%d/access_tokens", expectedInstall),
		func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Bearer test-jwt", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"token":      expectedToken,
				"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			})
		},
	)

	origJWTGen := gh_api.GenerateJWTToken
	origGHClientConstructor := gh_api.GHClientConstructor
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
		gh_api.GenerateJWTToken = origJWTGen
		gh_api.GHClientConstructor = origGHClientConstructor
	})
	defer server.Close()
	ghEnv := conf.GithubAPIEnvironment{
		AppID:          "5",
		PrivateKeyPath: tmpFile,
	}

	gh_api.GenerateJWTToken = func(privateKey *rsa.PrivateKey, appID string) (string, error) {
		return "test-jwt", nil
	}
	gh_api.GHClientConstructor = func(client *http.Client) *github.Client {
		cl, err := github.NewClient(client).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
		require.NoError(t, err)
		return cl
	}
	authorize, err := gh_api.Authorize(ghEnv, testOwner, testRepo)
	require.NoError(t, err)
	require.NotNil(t, authorize)
	require.Equal(t, expectedToken, *authorize.Token)
}
