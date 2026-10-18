package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/smford/tf-move-gen/internal/matcher"
)

// Options controls formatting and generation behavior.
type Options struct {
	Explain         bool
	StateMvCommands bool
	Append          bool
}

// Generate formats a slice of matches into HCL moved blocks or state mv commands.
func Generate(matches []matcher.Match, opts Options) (string, error) {
	if len(matches) == 0 {
		return "", nil
	}

	var sb strings.Builder
	for i, m := range matches {
		if i > 0 {
			sb.WriteString("\n")
		}
		if opts.StateMvCommands {
			sb.WriteString(FormatStateMv(m, opts.Explain))
		} else {
			sb.WriteString(FormatMovedBlock(m, opts.Explain))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// FormatExplanation generates a comment line showing confidence and matched attributes.
func FormatExplanation(m matcher.Match) string {
	pct := fmt.Sprintf("%.1f%%", m.Confidence*100)
	if len(m.MatchedAttributes) > 0 {
		return fmt.Sprintf("# Confidence: %s (matched on: %s)\n", pct, strings.Join(m.MatchedAttributes, ", "))
	}
	return fmt.Sprintf("# Confidence: %s\n", pct)
}

// FormatMovedBlock formats a single HCL moved block.
func FormatMovedBlock(m matcher.Match, explain bool) string {
	var sb strings.Builder
	if explain {
		sb.WriteString(FormatExplanation(m))
	}
	sb.WriteString("moved {\n")
	sb.WriteString(fmt.Sprintf("  from = %s\n", m.From))
	sb.WriteString(fmt.Sprintf("  to   = %s\n", m.To))
	sb.WriteString("}")
	return sb.String()
}

// FormatStateMv formats a single terraform state mv shell command.
func FormatStateMv(m matcher.Match, explain bool) string {
	var sb strings.Builder
	if explain {
		sb.WriteString(FormatExplanation(m))
	}
	fromEsc := shellEscape(m.From)
	toEsc := shellEscape(m.To)
	sb.WriteString(fmt.Sprintf("terraform state mv '%s' '%s'", fromEsc, toEsc))
	return sb.String()
}

func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

var (
	hclMovedRegex = regexp.MustCompile(`(?s)moved\s*\{[^}]*from\s*=\s*([^\s\r\n]+)[^}]*to\s*=\s*([^\s\r\n]+)[^}]*\}`)
	stateMvRegex  = regexp.MustCompile(`terraform\s+state\s+mv\s+'([^']+)'\s+'([^']+)'`)
)

// ParseExistingBlocks parses existing moved blocks or state mv commands from file content.
// Returns a map of from -> to.
func ParseExistingBlocks(content string) map[string]string {
	existing := make(map[string]string)

	hclMatches := hclMovedRegex.FindAllStringSubmatch(content, -1)
	for _, match := range hclMatches {
		if len(match) >= 3 {
			from := strings.TrimSpace(match[1])
			to := strings.TrimSpace(match[2])
			existing[from] = to
		}
	}

	mvMatches := stateMvRegex.FindAllStringSubmatch(content, -1)
	for _, match := range mvMatches {
		if len(match) >= 3 {
			from := strings.TrimSpace(match[1])
			to := strings.TrimSpace(match[2])
			existing[from] = to
		}
	}

	return existing
}

// FilterDuplicates filters out matches that already exist in the provided existing map.
func FilterDuplicates(matches []matcher.Match, existing map[string]string) []matcher.Match {
	if len(existing) == 0 {
		return matches
	}
	var filtered []matcher.Match
	for _, m := range matches {
		if existingTo, ok := existing[m.From]; ok && existingTo == m.To {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// WriteOutput writes or appends the generated output to outPath.
func WriteOutput(outPath string, matches []matcher.Match, opts Options) error {
	matchesToWrite := matches
	existingContent := ""

	if opts.Append {
		if data, err := os.ReadFile(outPath); err == nil {
			existingContent = string(data)
			existingMap := ParseExistingBlocks(existingContent)
			matchesToWrite = FilterDuplicates(matches, existingMap)
		}
	}

	if len(matchesToWrite) == 0 {
		// Nothing new to write/append
		if !opts.Append {
			// Write empty file
			return os.WriteFile(outPath, []byte(""), 0644)
		}
		return nil
	}

	newOutput, err := Generate(matchesToWrite, opts)
	if err != nil {
		return err
	}

	if opts.Append && existingContent != "" {
		if !strings.HasSuffix(existingContent, "\n") {
			existingContent += "\n"
		}
		if !strings.HasSuffix(existingContent, "\n\n") && !strings.HasSuffix(newOutput, "\n\n") {
			existingContent += "\n"
		}
		return os.WriteFile(outPath, []byte(existingContent+newOutput), 0644)
	}

	return os.WriteFile(outPath, []byte(newOutput), 0644)
}
