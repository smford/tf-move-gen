package matcher

import (
	"testing"

	"github.com/smford/tf-move-gen/internal/tfplan"
)

func TestDBInstanceMoveWithUnknownIDAndARN(t *testing.T) {
	// Scenario 1: An aws_db_instance moved into module.database.aws_db_instance.primary
	// matches with >0.95 confidence even when id and arn are unknown in after.
	del := &tfplan.ResourceChange{
		Address:      "aws_db_instance.primary",
		Mode:         "managed",
		Type:         "aws_db_instance",
		Name:         "primary",
		ProviderName: "registry.terraform.io/hashicorp/aws",
		Change: tfplan.Change{
			Actions: []string{"delete"},
			Before: map[string]any{
				"id":                "mydb-production",
				"arn":               "arn:aws:rds:us-east-1:123456789012:db:mydb-production",
				"identifier":        "mydb-production",
				"allocated_storage": float64(20),
				"engine":            "postgres",
				"engine_version":    "14.3",
				"instance_class":    "db.t4g.micro",
				"username":          "postgres",
				"tags": map[string]any{
					"Environment": "production",
					"Name":        "mydb-production",
				},
			},
			After: nil,
		},
	}

	create := &tfplan.ResourceChange{
		Address:      "module.database.aws_db_instance.primary",
		Mode:         "managed",
		Type:         "aws_db_instance",
		Name:         "primary",
		ProviderName: "registry.terraform.io/hashicorp/aws",
		Change: tfplan.Change{
			Actions: []string{"create"},
			Before:  nil,
			After: map[string]any{
				"identifier":        "mydb-production",
				"allocated_storage": float64(20),
				"engine":            "postgres",
				"engine_version":    "14.3",
				"instance_class":    "db.t4g.micro",
				"username":          "postgres",
				"tags": map[string]any{
					"Environment": "production",
					"Name":        "mydb-production",
				},
			},
			AfterUnknown: map[string]any{
				"id":       true,
				"arn":      true,
				"tags_all": true,
			},
		},
	}

	score, matchedAttrs := ComputeSimilarity(del, create)
	if score < 0.95 {
		t.Fatalf("expected confidence > 0.95, got %f", score)
	}

	// Verify plan matching
	plan := &tfplan.Plan{
		ResourceChanges: []tfplan.ResourceChange{*del, *create},
	}
	res, err := FindMatches(plan, Config{MinConfidence: 0.70})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(res.Matches))
	}
	m := res.Matches[0]
	if m.From != "aws_db_instance.primary" || m.To != "module.database.aws_db_instance.primary" {
		t.Errorf("unexpected match mapping: from=%s to=%s", m.From, m.To)
	}
	if len(matchedAttrs) == 0 {
		t.Errorf("expected matched attributes, got none")
	}
}

func TestSubnetCountToForEachMatching(t *testing.T) {
	// Scenario 2: Three aws_subnet resources converted from count ([0], [1], [2])
	// to for_each (["10.0.1.0/24"], etc.) match the exact corresponding CIDR blocks
	// rather than pairing randomly.
	cidrs := []string{"10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"}
	azs := []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"}

	var deletes []tfplan.ResourceChange
	var creates []tfplan.ResourceChange

	// count-based subnets [0], [1], [2]
	for i := 0; i < 3; i++ {
		deletes = append(deletes, tfplan.ResourceChange{
			Address:      "aws_subnet.public[" + string(rune('0'+i)) + "]",
			Mode:         "managed",
			Type:         "aws_subnet",
			Name:         "public",
			Index:        i,
			ProviderName: "registry.terraform.io/hashicorp/aws",
			Change: tfplan.Change{
				Actions: []string{"delete"},
				Before: map[string]any{
					"cidr_block":                  cidrs[i],
					"availability_zone":           azs[i],
					"vpc_id":                      "vpc-12345",
					"map_public_ip_on_launch":     true,
					"enable_resource_name_dns_a_record_on_launch": false,
				},
			},
		})
	}

	// for_each-based subnets in shuffled order: 2, 0, 1
	shuffledIndices := []int{2, 0, 1}
	for _, idx := range shuffledIndices {
		cidr := cidrs[idx]
		creates = append(creates, tfplan.ResourceChange{
			Address:      `aws_subnet.public["` + cidr + `"]`,
			Mode:         "managed",
			Type:         "aws_subnet",
			Name:         "public",
			Index:        cidr,
			ProviderName: "registry.terraform.io/hashicorp/aws",
			Change: tfplan.Change{
				Actions: []string{"create"},
				After: map[string]any{
					"cidr_block":                  cidr,
					"availability_zone":           azs[idx],
					"vpc_id":                      "vpc-12345",
					"map_public_ip_on_launch":     true,
					"enable_resource_name_dns_a_record_on_launch": false,
				},
				AfterUnknown: map[string]any{
					"id":  true,
					"arn": true,
				},
			},
		})
	}

	var allChanges []tfplan.ResourceChange
	allChanges = append(allChanges, deletes...)
	allChanges = append(allChanges, creates...)

	plan := &tfplan.Plan{ResourceChanges: allChanges}
	res, err := FindMatches(plan, Config{MinConfidence: 0.70})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(res.Matches))
	}
	if len(res.UnmatchedDeletes) != 0 {
		t.Errorf("expected 0 unmatched deletes, got %d", len(res.UnmatchedDeletes))
	}

	// Verify exact pairing: [0] <-> "10.0.1.0/24", [1] <-> "10.0.2.0/24", [2] <-> "10.0.3.0/24"
	expectedPairs := map[string]string{
		"aws_subnet.public[0]": `aws_subnet.public["10.0.1.0/24"]`,
		"aws_subnet.public[1]": `aws_subnet.public["10.0.2.0/24"]`,
		"aws_subnet.public[2]": `aws_subnet.public["10.0.3.0/24"]`,
	}

	for _, m := range res.Matches {
		expectedTo, ok := expectedPairs[m.From]
		if !ok {
			t.Fatalf("unexpected from address: %s", m.From)
		}
		if m.To != expectedTo {
			t.Errorf("for from %s: expected %s, got %s", m.From, expectedTo, m.To)
		}
		if m.Confidence < 0.95 {
			t.Errorf("expected confidence > 0.95 for exact match, got %f", m.Confidence)
		}
	}
}

func TestUnrelatedDeletesRemainUnmatched(t *testing.T) {
	// Scenario 3: Unrelated deletes remain unmatched when similarity is below threshold.
	del := tfplan.ResourceChange{
		Address:      "aws_s3_bucket.legacy_logs",
		Mode:         "managed",
		Type:         "aws_s3_bucket",
		Name:         "legacy_logs",
		ProviderName: "registry.terraform.io/hashicorp/aws",
		Change: tfplan.Change{
			Actions: []string{"delete"},
			Before: map[string]any{
				"bucket": "acme-legacy-app-logs-2021",
				"acl":    "private",
				"tags": map[string]any{
					"App": "legacy",
				},
			},
		},
	}

	create := tfplan.ResourceChange{
		Address:      "aws_s3_bucket.data_warehouse",
		Mode:         "managed",
		Type:         "aws_s3_bucket",
		Name:         "data_warehouse",
		ProviderName: "registry.terraform.io/hashicorp/aws",
		Change: tfplan.Change{
			Actions: []string{"create"},
			After: map[string]any{
				"bucket": "acme-corp-bi-warehouse-prod",
				"acl":    "public-read",
				"tags": map[string]any{
					"App": "warehouse",
				},
			},
		},
	}

	plan := &tfplan.Plan{
		ResourceChanges: []tfplan.ResourceChange{del, create},
	}

	res, err := FindMatches(plan, Config{MinConfidence: 0.70})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Matches) != 0 {
		t.Errorf("expected 0 matches for unrelated resources, got %d", len(res.Matches))
	}
	if len(res.UnmatchedDeletes) != 1 {
		t.Errorf("expected 1 unmatched delete, got %d", len(res.UnmatchedDeletes))
	}
	if len(res.UnmatchedCreates) != 1 {
		t.Errorf("expected 1 unmatched create, got %d", len(res.UnmatchedCreates))
	}
}

func TestTypeIsolation(t *testing.T) {
	// Candidate resources of different types should never be matched
	del := tfplan.ResourceChange{
		Address:      "aws_s3_bucket.main",
		Mode:         "managed",
		Type:         "aws_s3_bucket",
		Name:         "main",
		ProviderName: "registry.terraform.io/hashicorp/aws",
		Change: tfplan.Change{
			Actions: []string{"delete"},
			Before: map[string]any{
				"name": "my-resource",
			},
		},
	}

	create := tfplan.ResourceChange{
		Address:      "aws_iam_role.main",
		Mode:         "managed",
		Type:         "aws_iam_role",
		Name:         "main",
		ProviderName: "registry.terraform.io/hashicorp/aws",
		Change: tfplan.Change{
			Actions: []string{"create"},
			After: map[string]any{
				"name": "my-resource",
			},
		},
	}

	plan := &tfplan.Plan{
		ResourceChanges: []tfplan.ResourceChange{del, create},
	}

	res, err := FindMatches(plan, Config{MinConfidence: 0.50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Matches) != 0 {
		t.Errorf("expected 0 matches between different resource types, got %d", len(res.Matches))
	}
}
