# Getting Started with tf-move-gen

This guide walks you through a hands-on refactoring scenario, showing you how to turn destructive plan recreations into seamless zero-downtime migrations.

---

## Prerequisites

1. **Go 1.23+** (or download the precompiled binary from [GitHub Releases](https://github.com/smford/tf-move-gen/releases)).
2. **Terraform 1.1+** or **OpenTofu 1.6+**.
3. An existing Terraform configuration with resources already deployed.

### Installation

```bash
# Install via Go
go install github.com/smford/tf-move-gen/cmd/tf-move-gen@latest

# Verify installation
tf-move-gen --help
```

---

## Walkthrough: Extracting Resources into a Module

Consider an existing monolithic `main.tf` with an S3 bucket and an IAM role:

```hcl
# main.tf (Before Refactoring)
resource "aws_s3_bucket" "media" {
  bucket = "acme-corp-media-assets"
  acl    = "private"

  tags = {
    Environment = "production"
    Project     = "media-pipeline"
  }
}

resource "aws_iam_role" "worker" {
  name = "acme-worker-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = []
  })
}
```

### Step 1: Perform Your Code Refactoring

You create a new child module `modules/storage` and move the bucket and role definitions into it:

```hcl
# main.tf (After Refactoring)
module "storage" {
  source = "./modules/storage"
}
```

```hcl
# modules/storage/main.tf
resource "aws_s3_bucket" "media" {
  bucket = "acme-corp-media-assets"
  acl    = "private"

  tags = {
    Environment = "production"
    Project     = "media-pipeline"
  }
}

resource "aws_iam_role" "worker" {
  name = "acme-worker-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = []
  })
}
```

### Step 2: See the Destructive Plan

If you run `terraform plan`, Terraform will report:

```text
Plan: 2 to add, 0 to change, 2 to destroy.

Do you really want to destroy your production S3 bucket? ❌
```

### Step 3: Run tf-move-gen

Instead of writing `moved` blocks by hand:

```bash
# 1. Output the plan file
terraform plan -out=tfplan

# 2. Pipe the plan JSON into tf-move-gen
terraform show -json tfplan | tf-move-gen --explain --out=moves.tf
```

`tf-move-gen` inspects the planned deletes and creates, compares their configuration attributes, and writes `moves.tf`:

```hcl
# Confidence: 100.0% (matched on: acl, bucket, tags)
moved {
  from = aws_s3_bucket.media
  to   = module.storage.aws_s3_bucket.media
}

# Confidence: 100.0% (matched on: assume_role_policy, name)
moved {
  from = aws_iam_role.worker
  to   = module.storage.aws_iam_role.worker
}
```

### Step 4: Verify the Safe Plan

Re-run `terraform plan`:

```text
module.storage.aws_s3_bucket.media has moved to aws_s3_bucket.media
module.storage.aws_iam_role.worker has moved to aws_iam_role.worker

No changes. Your infrastructure matches the configuration. 🎉
```

Now you can apply or commit `moves.tf` directly to your repository!

---

## Scenario 2: Converting `count` to `for_each`

When refactoring subnets from numeric index to string CIDR keys:

```hcl
# Before:
resource "aws_subnet" "public" {
  count      = 3
  cidr_block = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"][count.index]
}

# After:
resource "aws_subnet" "public" {
  for_each   = toset(["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"])
  cidr_block = each.key
}
```

Run `tf-move-gen`:

```bash
terraform show -json tfplan | tf-move-gen --explain
```

Output:

```hcl
# Confidence: 100.0% (matched on: cidr_block, vpc_id)
moved {
  from = aws_subnet.public[0]
  to   = aws_subnet.public["10.0.1.0/24"]
}

# Confidence: 100.0% (matched on: cidr_block, vpc_id)
moved {
  from = aws_subnet.public[1]
  to   = aws_subnet.public["10.0.2.0/24"]
}

# Confidence: 100.0% (matched on: cidr_block, vpc_id)
moved {
  from = aws_subnet.public[2]
  to   = aws_subnet.public["10.0.3.0/24"]
}
```

`tf-move-gen` matches each CIDR block individually, ensuring subnets are never swapped or randomized.

---

## Scenario 3: Legacy State Move Commands

If you are using Terraform versions earlier than 1.1, or performing surgical cross-state migrations between remote backends, use `--state-mv-commands`:

```bash
terraform show -json tfplan | tf-move-gen --state-mv-commands
```

Output:

```bash
terraform state mv 'aws_subnet.public[0]' 'aws_subnet.public["10.0.1.0/24"]'
terraform state mv 'aws_subnet.public[1]' 'aws_subnet.public["10.0.2.0/24"]'
terraform state mv 'aws_subnet.public[2]' 'aws_subnet.public["10.0.3.0/24"]'
```

---

## Appending to Existing Moves Files

When doing large iterative refactors across multiple plan cycles, use the `--append` flag:

```bash
terraform show -json tfplan | tf-move-gen --append --out=moves.tf
```

`tf-move-gen` parses `moves.tf`, detects any existing `moved` blocks, and only appends new, non-duplicate moves.

---

## Next Steps

- Learn about [System Architecture](architecture.md).
- Read the [Matching Algorithm & Heuristics](matching-algorithm.md) breakdown.
- Explore the [CLI Reference](cli-reference.md).
- Set up automated [CI/CD Integration](ci-cd-integration.md).
