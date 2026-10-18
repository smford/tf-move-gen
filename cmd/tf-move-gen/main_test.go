package main

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIModuleExtractionFile(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "module_extraction.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--explain", fixturePath}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}

	out := stdout.String()
	// Should have matched media bucket and worker role
	if !strings.Contains(out, "from = aws_s3_bucket.media") || !strings.Contains(out, "to   = module.storage.aws_s3_bucket.media") {
		t.Errorf("missing media bucket moved block:\n%s", out)
	}
	if !strings.Contains(out, "from = aws_iam_role.worker") || !strings.Contains(out, "to   = module.compute.aws_iam_role.worker") {
		t.Errorf("missing worker iam role moved block:\n%s", out)
	}

	// Should have explain comments
	if !strings.Contains(out, "# Confidence:") {
		t.Errorf("missing confidence comment in output:\n%s", out)
	}

	// Should warn about unmatched aws_security_group.obsolete_bastion on stderr
	errOut := stderr.String()
	if !strings.Contains(errOut, "[WARNING] Unmatched delete: aws_security_group.obsolete_bastion") {
		t.Errorf("expected warning for unmatched delete on stderr, got:\n%s", errOut)
	}
}

func TestCLIStdinPiping(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "resource_renaming.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed reading fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-"}, bytes.NewReader(data), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "from = aws_dynamodb_table.old_table") || !strings.Contains(out, "to   = aws_dynamodb_table.new_table") {
		t.Errorf("missing dynamodb table moved block:\n%s", out)
	}
}

func TestCLIStateMvCommands(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "count_to_for_each.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--state-mv-commands", fixturePath}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}

	out := stdout.String()
	expectedCmd := `terraform state mv 'aws_subnet.public[0]' 'aws_subnet.public["10.0.1.0/24"]'`
	if !strings.Contains(out, expectedCmd) {
		t.Errorf("missing expected state mv command in:\n%s", out)
	}
}

func TestCLIFailOnUnmatchedDelete(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "module_extraction.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// fixture has aws_security_group.obsolete_bastion which has no matching create
	code := run([]string{"--fail-on-unmatched-delete", fixturePath}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2 on unmatched delete, got %d", code)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "destructive resource deletion(s) remain unmatched") {
		t.Errorf("expected error message on stderr, got:\n%s", errOut)
	}
}

func TestCLIOutFileAndAppend(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "moves.tf")
	fixture1 := filepath.Join("..", "..", "testdata", "resource_renaming.json")
	fixture2 := filepath.Join("..", "..", "testdata", "count_to_for_each.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// First run to write outFile
	code := run([]string{"--out", outFile, fixture1}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("first run failed with code %d: %s", code, stderr.String())
	}

	content1, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed reading out file: %v", err)
	}
	if !strings.Contains(string(content1), "aws_dynamodb_table.old_table") {
		t.Errorf("outFile missing first match")
	}

	// Append run
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--out", outFile, "--append", fixture2}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("append run failed with code %d: %s", code, stderr.String())
	}

	content2, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed reading out file after append: %v", err)
	}
	if !strings.Contains(string(content2), "aws_dynamodb_table.old_table") {
		t.Errorf("outFile lost first match after append")
	}
	if !strings.Contains(string(content2), "aws_subnet.public[0]") {
		t.Errorf("outFile missing appended subnet match")
	}
}

func TestCLIHealthz(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Run health check server in goroutine
	addr := "127.0.0.1:28394"
	go func() {
		_ = run([]string{"--serve-healthz", addr}, nil, &stdout, &stderr)
	}()

	// Query /healthz
	client := &http.Client{Timeout: 2 * time.Second}
	var resp *http.Response
	var err error
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		resp, err = client.Get("http://" + addr + "/healthz")
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("failed to connect to /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestCLIVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "tf-move-gen") || !strings.Contains(out, "dev") {
		t.Errorf("unexpected version output: %s", out)
	}
}
