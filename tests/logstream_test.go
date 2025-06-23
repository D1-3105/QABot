package tests

import (
	"ActQABot/api/github_api"
	"bufio"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogStreamer(t *testing.T) {

	grpcConnFixture(t)
	setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/job/logs/?host=my-vm&job_id=abc", nil)
	w := httptest.NewRecorder()

	router := github_api.Router()
	router.ServeHTTP(w, req)

	res := w.Result()
	defer func(Body io.ReadCloser) {
		t.Log("response body closed")
		_ = Body.Close()
	}(res.Body)

	require.Equal(t, http.StatusOK, res.StatusCode)

	scanner := bufio.NewScanner(res.Body)
	lineTimeout := 10 * time.Second
	lines := []string{}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), lineTimeout)
		lineCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			if scanner.Scan() {
				lineCh <- scanner.Text()
			} else {
				if err := scanner.Err(); err != nil {
					errCh <- err
				} else {
					errCh <- io.EOF
				}
			}
		}()

		select {
		case line := <-lineCh:
			lines = append(lines, line)
			t.Logf("received: %s", line)

		case err := <-errCh:
			cancel()
			if err == io.EOF {
				t.Log("log stream ended")
				goto DONE
			}
			t.Fatalf("error while reading: %v", err)

		case <-ctx.Done():
			cancel()
			t.Fatalf("timeout waiting for next line")
		}
		cancel()
	}

DONE:
	require.GreaterOrEqual(t, len(lines), 2, "should receive at least 2 log lines")
	assert.Contains(t, lines[0], "line1")
	assert.Contains(t, lines[1], "line2")
}
