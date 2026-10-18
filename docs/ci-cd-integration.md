# CI/CD Integration & Safety Gates

One of the most powerful use cases for `tf-move-gen` is establishing an automated **refactoring safety gate** in CI/CD pipelines.

---

## Why Use tf-move-gen in CI?

During team refactorings, pull requests often modify module paths, re-structure directories, or rename resources. If an engineer forgets a single `moved` block:
- The pull request plan may contain hundreds of lines of output.
- Reviewers suffering from review fatigue might overlook a critical database or subnet deletion buried in the diff.
- Merging the PR immediately triggers destructive changes upon deployment.

By running `tf-move-gen --fail-on-unmatched-delete`, your CI pipeline acts as an automated safety firewall: it automatically detects if any proposed deletion lacks an intentional matching target.

---

## GitHub Actions: PR Refactoring Safety Gate

Below is a complete, production-grade GitHub Actions workflow that plans infrastructure changes, verifies that all refactored resources have valid moves, and comments on the pull request:

```yaml
name: Terraform Refactor Safety Gate

on:
  pull_request:
    branches: [main]

permissions:
  contents: read
  pull-requests: write

jobs:
  plan-and-verify:
    name: Verify Refactor Moves
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3

      - name: Setup tf-move-gen
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go install github.com/smford/tf-move-gen/cmd/tf-move-gen@latest

      - name: Terraform Init
        run: terraform init

      - name: Generate Terraform Plan
        run: terraform plan -out=tfplan

      - name: Verify Moves & Enforce Safety Gate
        id: move-check
        run: |
          terraform show -json tfplan | tf-move-gen \
            --explain \
            --out=proposed-moves.tf \
            --fail-on-unmatched-delete

      - name: Post Proposed Moves to Pull Request
        if: always() && steps.move-check.outcome == 'failure'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            let body = "### ⚠️ Terraform Refactor Gate Alert\n\n";
            body += "Destructive deletions were detected that could not be safely matched by `tf-move-gen`.\n";
            if (fs.existsSync('proposed-moves.tf')) {
              const moves = fs.readFileSync('proposed-moves.tf', 'utf8');
              body += "\nProposed matching moves generated:\n```hcl\n" + moves + "\n```";
            }
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
```

---

## Automated Refactoring Bot

You can also use `tf-move-gen` to automatically generate `moves.tf` and commit it directly to the refactoring PR branch:

```yaml
      - name: Auto-Commit Generated Moves
        if: success()
        run: |
          terraform show -json tfplan | tf-move-gen --append --out=moves.tf
          if [ -n "$(git status --porcelain moves.tf)" ]; then
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            git add moves.tf
            git commit -m "chore(terraform): auto-generate moved blocks via tf-move-gen"
            git push
          fi
```

---

## GitLab CI / Jenkins

In standard shell pipelines:

```bash
# Generate plan JSON
terraform plan -out=tfplan
terraform show -json tfplan > plan.json

# Run gate: exits with code 2 if unmatched deletes exist
tf-move-gen --fail-on-unmatched-delete --explain plan.json
```
