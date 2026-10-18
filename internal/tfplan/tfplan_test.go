package tfplan

import (
	"strings"
	"testing"
)

func TestParseBytes(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.8.0",
		"resource_changes": [
			{
				"address": "aws_s3_bucket.old",
				"mode": "managed",
				"type": "aws_s3_bucket",
				"name": "old",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["delete"],
					"before": {"bucket": "my-bucket", "acl": "private"},
					"after": null
				}
			},
			{
				"address": "module.storage.aws_s3_bucket.new",
				"mode": "managed",
				"type": "aws_s3_bucket",
				"name": "new",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"bucket": "my-bucket", "acl": "private"},
					"after_unknown": {"id": true, "arn": true}
				}
			},
			{
				"address": "data.aws_caller_identity.current",
				"mode": "data",
				"type": "aws_caller_identity",
				"name": "current",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["read"]
				}
			},
			{
				"address": "aws_iam_role.existing",
				"mode": "managed",
				"type": "aws_iam_role",
				"name": "existing",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["no-op"]
				}
			}
		]
	}`

	plan, err := ParseBytes([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.FormatVersion != "1.2" {
		t.Errorf("expected format_version 1.2, got %s", plan.FormatVersion)
	}
	if len(plan.ResourceChanges) != 4 {
		t.Fatalf("expected 4 resource changes, got %d", len(plan.ResourceChanges))
	}

	deletes, creates := plan.Candidates()
	if len(deletes) != 1 {
		t.Errorf("expected 1 delete candidate, got %d", len(deletes))
	}
	if deletes[0].Address != "aws_s3_bucket.old" {
		t.Errorf("expected delete address aws_s3_bucket.old, got %s", deletes[0].Address)
	}

	if len(creates) != 1 {
		t.Errorf("expected 1 create candidate, got %d", len(creates))
	}
	if creates[0].Address != "module.storage.aws_s3_bucket.new" {
		t.Errorf("expected create address module.storage.aws_s3_bucket.new, got %s", creates[0].Address)
	}
}

func TestEmptyPlan(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error on empty input, got nil")
	}
}

func TestProviderBaseName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"aws", "aws"},
		{"hashicorp/aws", "aws"},
		{"registry.terraform.io/hashicorp/aws", "aws"},
	}
	for _, tc := range cases {
		got := ProviderBaseName(tc.input)
		if got != tc.expected {
			t.Errorf("ProviderBaseName(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}
