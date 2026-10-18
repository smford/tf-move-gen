package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smford/tf-move-gen/internal/matcher"
)

func TestGenerateHCL(t *testing.T) {
	matches := []matcher.Match{
		{
			From:              "aws_s3_bucket.old",
			To:                "module.storage.aws_s3_bucket.new",
			Confidence:        0.964,
			MatchedAttributes: []string{"acl", "bucket", "tags"},
		},
	}

	// Without explain
	out, err := Generate(matches, Options{Explain: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "moved {\n  from = aws_s3_bucket.old\n  to   = module.storage.aws_s3_bucket.new\n}\n"
	if out != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, out)
	}

	// With explain
	outExplain, err := Generate(matches, Options{Explain: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(outExplain, "# Confidence: 96.4% (matched on: acl, bucket, tags)") {
		t.Errorf("expected explanation comment, got:\n%s", outExplain)
	}
}

func TestGenerateStateMv(t *testing.T) {
	matches := []matcher.Match{
		{
			From:              "aws_subnet.public[0]",
			To:                `aws_subnet.public["eu-west-2a"]`,
			Confidence:        1.0,
			MatchedAttributes: []string{"cidr_block", "vpc_id"},
		},
	}

	out, err := Generate(matches, Options{StateMvCommands: true, Explain: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrefix := "# Confidence: 100.0% (matched on: cidr_block, vpc_id)\n"
	expectedCmd := `terraform state mv 'aws_subnet.public[0]' 'aws_subnet.public["eu-west-2a"]'`
	if !strings.HasPrefix(out, expectedPrefix) {
		t.Errorf("expected prefix:\n%s\ngot:\n%s", expectedPrefix, out)
	}
	if !strings.Contains(out, expectedCmd) {
		t.Errorf("expected command:\n%s\ngot:\n%s", expectedCmd, out)
	}
}

func TestWriteOutputAndAppendDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "moves.tf")

	initialMatch := []matcher.Match{
		{
			From:       "aws_s3_bucket.first",
			To:         "module.s3.aws_s3_bucket.first",
			Confidence: 1.0,
		},
	}

	// First write
	err := WriteOutput(outFile, initialMatch, Options{})
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading file failed: %v", err)
	}
	if !strings.Contains(string(content), "aws_s3_bucket.first") {
		t.Errorf("missing first match in file")
	}

	// Append with duplicate first match and new second match
	secondBatch := []matcher.Match{
		{
			From:       "aws_s3_bucket.first",
			To:         "module.s3.aws_s3_bucket.first",
			Confidence: 1.0,
		},
		{
			From:       "aws_iam_role.app",
			To:         "module.iam.aws_iam_role.app",
			Confidence: 0.95,
		},
	}

	err = WriteOutput(outFile, secondBatch, Options{Append: true})
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}

	finalContent, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading file after append failed: %v", err)
	}

	// Verify only one occurrence of aws_s3_bucket.first
	countFirst := strings.Count(string(finalContent), "aws_s3_bucket.first")
	// Note: in moved block, "aws_s3_bucket.first" appears twice (from = ..., to = module.s3.aws_s3_bucket.first)
	if countFirst != 2 {
		t.Errorf("expected exactly 2 occurrences of 'aws_s3_bucket.first' (1 block), got %d:\n%s", countFirst, string(finalContent))
	}

	if !strings.Contains(string(finalContent), "aws_iam_role.app") {
		t.Errorf("expected appended file to contain aws_iam_role.app")
	}
}
