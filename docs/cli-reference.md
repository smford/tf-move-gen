# CLI Reference

`tf-move-gen` provides a command-line interface designed for terminal workflows, shell piping, and automated CI/CD safety checks.

---

## Synopsis

```bash
tf-move-gen [flags] [plan.json]
```

If `[plan.json]` is omitted or specified as `-`, `tf-move-gen` reads the plan JSON from standard input (`stdin`).

---

## Flags

### `--min-confidence <float>`
- **Type:** Floating point (`0.0` to `1.0`)
- **Default:** `0.70`
- **Description:** Sets the minimum weighted similarity threshold required for a delete-create pair to be matched. Pairs below this threshold remain unmatched and will trigger warnings in `--explain` mode.

```bash
# Require 90% confidence for matches
tf-move-gen --min-confidence 0.90 plan.json
```

---

### `--out <file.tf>`
- **Type:** File path string
- **Default:** `""` (prints to `stdout`)
- **Description:** Writes the generated HCL `moved` blocks (or `state mv` commands) directly to the specified file path (e.g., `moves.tf`).

```bash
tf-move-gen --out moves.tf plan.json
```

---

### `--append`
- **Type:** Boolean
- **Default:** `false`
- **Description:** When used with `--out`, appends new moves to an existing file rather than overwriting it. It parses the existing file and automatically deduplicates any blocks that are already present.

```bash
tf-move-gen --out moves.tf --append plan.json
```

---

### `--explain`
- **Type:** Boolean
- **Default:** `false`
- **Description:** Includes human-readable comments above each generated block with the exact confidence percentage and list of matched attributes. Also emits warnings to `stderr` for any unmatched deletions.

```bash
tf-move-gen --explain plan.json
```

Example output:
```hcl
# Confidence: 100.0% (matched on: bucket, force_destroy, tags)
moved {
  from = aws_s3_bucket.media
  to   = module.storage.aws_s3_bucket.media
}
```

---

### `--state-mv-commands`
- **Type:** Boolean
- **Default:** `false`
- **Description:** Outputs legacy `terraform state mv '<from>' '<to>'` shell commands instead of HCL `moved` blocks. Useful for Terraform versions prior to 1.1 or cross-state surgical migrations.

```bash
tf-move-gen --state-mv-commands plan.json
```

Example output:
```bash
terraform state mv 'aws_subnet.public[0]' 'aws_subnet.public["10.0.1.0/24"]'
```

---

### `--fail-on-unmatched-delete`
- **Type:** Boolean
- **Default:** `false`
- **Description:** Causes `tf-move-gen` to exit with status code `2` if any destructive deletion in the plan remains unmatched. This is designed for pull request gates in CI/CD pipelines to prevent unapproved resource destruction.

```bash
terraform show -json tfplan | tf-move-gen --fail-on-unmatched-delete --out moves.tf
```

---

### `--serve-healthz <addr>`
- **Type:** String (e.g. `:8080`)
- **Default:** `""`
- **Description:** Starts a lightweight HTTP daemon listening on the specified address with a `GET /healthz` endpoint. Used for container health verification and DAST security scans.

```bash
tf-move-gen --serve-healthz :8080
```

---

## Exit Codes

| Exit Code | Meaning |
| :---: | :--- |
| `0` | **Success**: Plan processed, matches generated (or no resources needed moving). |
| `1` | **Error**: Malformed arguments, invalid JSON, or file I/O failure. |
| `2` | **Safety Violation**: Unmatched destructive deletions detected when `--fail-on-unmatched-delete` is enabled. |

---

## Common Recipes

### One-Liner Plan & Move Generation
```bash
terraform plan -out=tfplan && terraform show -json tfplan | tf-move-gen --explain --out=moves.tf
```

### Dry-Run Preview in Terminal
```bash
terraform show -json tfplan | tf-move-gen --explain
```

### CI Pull Request Gate
```bash
terraform show -json tfplan | tf-move-gen --fail-on-unmatched-delete --explain --out=moves.tf
```
