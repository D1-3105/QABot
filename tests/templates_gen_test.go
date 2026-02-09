package tests

import (
	"ActQABot/templates"
	"strings"
	"testing"
)

func TestHelpCmdContext_GenText(t *testing.T) {
	setupTestEnv(t)
	oldText := []string{
		"Old text 1",
		"Old text 2",
	}
	helpCommand := "/help"
	supportedCommands := []string{"/help", "/start", "/status"}
	startJob := templates.StartJobHelpContext{StartCommand: "/startjob"}

	ctx := templates.NewHelpCmdContext(oldText, helpCommand, supportedCommands, startJob)

	out, err := ctx.GenText()
	if err != nil {
		t.Fatalf("GenText failed: %v", err)
	}
	t.Logf("Generated output:\n%s", out)

	if !strings.Contains(out, helpCommand) {
		t.Errorf("output does not contain HelpCommand: %q", helpCommand)
	}

	found := false
	for _, cmd := range supportedCommands {
		if strings.Contains(out, cmd) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("output does not contain any supported commands: %v", supportedCommands)
	}
	for _, old := range oldText {
		if !strings.Contains(out, old) {
			t.Errorf("output does not contain old text: %q", old)
		}
	}
}

func TestErrorContext_GenText(t *testing.T) {
	setupTestEnv(t)
	ctx := templates.NewErrorResultContext("Error message")
	out, err := ctx.GenText()
	if err != nil {
		t.Fatalf("GenText failed: %v", err)
	}
	t.Logf("Generated output:\n%s", out)
	if !strings.Contains(out, "Error message") {
		t.Errorf("output does not contain error message: %q", "Error message")
	}
}

func TestWorkerReport_GenText(t *testing.T) {
	setupTestEnv(t)
	useContext := templates.NewWorkerReportContext(
		"@bot /wf_start myvm 23f0ef6ea449131defae72433a2b7600e5158e83 .github/workflows/bench-qwen_image.yaml",
		`
BeepBoop: new job started
Log tracking url: https://my-uri/job/logs?host=myvm&job_id=25a33a0b-02c9-425f-a135-7a10216a861d

Detected Docker Environment:

 -e  TEST_CASE=...
`,
		`
| Item      | Quantity | Price  |
|-----------|---------|--------|
| Apple     | 10      | $2.00  |
| Banana    | 5       | $1.50  |
| Orange    | 8       | $2.50  |
`,
	)
	out, err := useContext.GenText()
	if err != nil {
		t.Fatalf("GenText failed: %v", err)
	}
	t.Logf("Generated output:\n%s", out)
	if !strings.Contains(out, "@bot") {
		t.Errorf("output does not contain Initial")
	}
	if !strings.Contains(out, "BeepBoop: new job started") {
		t.Errorf("output does not contain Answer")
	}
	if !strings.Contains(out, "Orange") {
		t.Errorf("output does not contain Report")
	}
}
