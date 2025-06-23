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
