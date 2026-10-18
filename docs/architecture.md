# System Architecture

`tf-move-gen` is engineered as a zero-CGO, single-binary Go utility that operates as a pure data-transformation pipeline on Terraform and OpenTofu execution plan JSON.

---

## High-Level Pipeline

```mermaid
flowchart TD
    subgraph Input["Input Stage"]
        JSON[("Terraform JSON Plan\n(File or Stdin)")]
    end

    subgraph Parser["Package internal/tfplan"]
        P1["JSON Plan Parser"]
        P2["Candidate Extractor\n- Pure Deletes\n- Pure Creates\n- Ignore Data Sources"]
        P3["Group by Type & Provider\n(e.g., aws_db_instance)"]
    end

    subgraph Matcher["Package internal/matcher"]
        M1["Attribute Flattener\n(Dot notation & List indexing)"]
        M2["Computed Field Stripper\n- after_unknown\n- Transient IDs & ARNs"]
        M3["Weighted Similarity Scorer\n- 2.0x high-signal keys\n- Slices Jaccard overlap"]
        M4["Greedy Bipartite Matcher\nPairs above --min-confidence"]
    end

    subgraph Generator["Package internal/generator"]
        G1["HCL Moved Formatter\n(Correct raw expressions & quotes)"]
        G2["State Mv Formatter\n(Shell-escaped commands)"]
        G3["Deduplicator Engine\n(--append file merging)"]
    end

    subgraph Output["Output Stage"]
        OutFile["moves.tf / stdout"]
        ExitCode["Exit Code 0 (clean) / 2 (unmatched)"]
    end

    JSON --> P1
    P1 --> P2
    P2 --> P3
    P3 --> M1
    M1 --> M2
    M2 --> M3
    M3 --> M4
    M4 --> G1
    M4 --> G2
    G1 --> G3
    G2 --> G3
    G3 --> OutFile
    M4 --> ExitCode
```

---

## Package Breakdown

### 1. `internal/tfplan` — Plan Parser & Candidate Extractor

The `tfplan` package deserializes the output of `terraform show -json` or `tofu show -json`.

- **Candidate Isolation:** Inspects each entry in `resource_changes`. A resource change is considered a candidate source if its mode is `managed` and its actions are `["delete"]` (or a replacement where addresses differ). A resource is considered a candidate target if its actions are `["create"]`.
- **Mode Filtering:** Data sources (`mode == "data"`) and unchanged resources (`["no-op"]` or `["read"]`) are excluded from matching.
- **Provider Normalization:** Providers like `registry.terraform.io/hashicorp/aws` and `aws` are normalized to their base names to ensure compatibility across modules and registry prefixes.

### 2. `internal/matcher` — Similarity Engine & Bipartite Matcher

The `matcher` package pairs candidate deletions with candidate creations.

- **Group Partitioning:** Candidates are strictly grouped by `(ResourceType, Provider)`. Resources of differing types (e.g., `aws_s3_bucket` and `aws_iam_role`) are never compared.
- **Flattening:** Nested maps and block slices are recursively flattened into dot-separated paths (e.g., `tags.Environment`, `versioning[0].enabled`).
- **Computed Field Stripping:** Suppresses attributes present in `after_unknown` and known transient provider attributes (`id`, `arn`, `etag`, `unique_id`, `create_date`).
- **Weighted Jaccard Scoring:** Evaluates configuration equality with 2.0x weight for identifying attributes (`name`, `bucket`, `identifier`, `cidr_block`, `tags.Name`).
- **Greedy Bipartite Matching:** Computes all pairwise scores in each group, sorts candidates descending by score, and greedily selects the best non-overlapping matches above `--min-confidence`.

### 3. `internal/generator` — Code Generation & Deduplication

The `generator` package turns matched pairs into actionable code:

- **HCL Formatting:** Generates clean HCL `moved` blocks. Terraform requires `from` and `to` to be unquoted raw expressions, while preserving internal string keys (e.g. `aws_subnet.public["10.0.1.0/24"]`).
- **State MV Commands:** When `--state-mv-commands` is requested, formats shell-escaped `terraform state mv '<from>' '<to>'` commands.
- **Deduplication:** When `--append` is specified, the generator parses existing moved blocks in the target file and ensures duplicate moves are never inserted.

---

## State Diagram: Matching Lifecycle

```mermaid
stateDiagram-v2
    [*] --> LoadPlan: Read JSON
    LoadPlan --> ExtractCandidates: Filter managed delete/create
    ExtractCandidates --> GroupCandidates: Partition by (Type, Provider)
    GroupCandidates --> PairwiseScore: Compute weighted attribute similarity
    PairwiseScore --> FilterThreshold: Compare against min-confidence
    FilterThreshold --> BipartiteSort: Order pairs by score descending
    BipartiteSort --> GreedyPairing: Assign optimal unique pairs
    GreedyPairing --> GenerateOutput: Render HCL or State Mv
    GenerateOutput --> [*]
```

---

## Design Principles

1. **Zero External Dependencies at Runtime:** Standard Go libraries only. Compiles with `CGO_ENABLED=0` to a static binary.
2. **100% Offline Execution:** Never dials AWS, Azure, GCP, or HashiCorp APIs. All logic is derived from the plan JSON.
3. **Safety First:** If a resource deletion cannot be matched with confidence, it is surfaced as a warning. In CI, `--fail-on-unmatched-delete` halts the pipeline with exit code 2.
