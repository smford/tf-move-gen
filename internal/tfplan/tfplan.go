package tfplan

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// Plan represents the top-level structure of a Terraform JSON plan.
type Plan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	ResourceChanges  []ResourceChange `json:"resource_changes"`
}

// ResourceChange represents a single resource change entry in the plan.
type ResourceChange struct {
	Address         string `json:"address"`
	PreviousAddress string `json:"previous_address,omitempty"`
	ModuleAddress   string `json:"module_address,omitempty"`
	Mode            string `json:"mode"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	Index           any    `json:"index,omitempty"`
	ProviderName    string `json:"provider_name"`
	Change          Change `json:"change"`
}

// Change represents the detailed change object inside a ResourceChange.
type Change struct {
	Actions         []string       `json:"actions"`
	Before          map[string]any `json:"before"`
	After           map[string]any `json:"after"`
	AfterUnknown    map[string]any `json:"after_unknown"`
	BeforeSensitive any            `json:"before_sensitive"`
	AfterSensitive  any            `json:"after_sensitive"`
}

// Parse reads JSON plan data from an io.Reader and unmarshals it into a Plan.
func Parse(r io.Reader) (*Plan, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses Terraform plan JSON bytes.
func ParseBytes(data []byte) (*Plan, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("empty plan input")
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parsing terraform plan JSON: %w", err)
	}
	return &plan, nil
}

// ParseFile parses a Terraform plan JSON file from disk.
func ParseFile(filename string) (*Plan, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening plan file: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

// IsManaged returns true if the resource mode is managed (not data source).
func (rc *ResourceChange) IsManaged() bool {
	// mode is usually "managed" or "data"
	if rc.Mode == "data" {
		return false
	}
	return true
}

// IsDelete returns true if this change represents a resource deletion.
func (rc *ResourceChange) IsDelete() bool {
	if !rc.IsManaged() {
		return false
	}
	actions := rc.Change.Actions
	if len(actions) == 1 && actions[0] == "delete" {
		return true
	}
	// Replacement where address changed
	if rc.PreviousAddress != "" && rc.PreviousAddress != rc.Address {
		if slices.Contains(actions, "delete") && slices.Contains(actions, "create") {
			return true
		}
	}
	return false
}

// IsCreate returns true if this change represents a resource creation.
func (rc *ResourceChange) IsCreate() bool {
	if !rc.IsManaged() {
		return false
	}
	actions := rc.Change.Actions
	if len(actions) == 1 && actions[0] == "create" {
		return true
	}
	return false
}

// Candidates returns all candidate delete and create resource changes.
func (p *Plan) Candidates() (deletes []*ResourceChange, creates []*ResourceChange) {
	for i := range p.ResourceChanges {
		rc := &p.ResourceChanges[i]
		if rc.IsDelete() {
			deletes = append(deletes, rc)
		}
		if rc.IsCreate() {
			creates = append(creates, rc)
		}
	}
	return deletes, creates
}

// ProviderBaseName extracts the simple provider identifier (e.g. "aws" from "registry.terraform.io/hashicorp/aws").
func ProviderBaseName(provider string) string {
	parts := strings.Split(provider, "/")
	return parts[len(parts)-1]
}
