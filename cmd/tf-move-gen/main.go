package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/smford/tf-move-gen/internal/generator"
	"github.com/smford/tf-move-gen/internal/matcher"
	"github.com/smford/tf-move-gen/internal/tfplan"
)

func main() {
	exitCode := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tf-move-gen", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		minConfidence          float64
		outFile                string
		appendOut              bool
		explain                bool
		stateMvCommands        bool
		failOnUnmatchedDelete bool
	)

	fs.Float64Var(&minConfidence, "min-confidence", matcher.DefaultMinConfidence, "Minimum similarity threshold between 0.0 and 1.0")
	fs.StringVar(&outFile, "out", "", "Write generated HCL blocks directly to a file")
	fs.BoolVar(&appendOut, "append", false, "Append to --out file if it already exists, deduplicating existing moved blocks")
	fs.BoolVar(&explain, "explain", false, "Include comments above each moved block showing confidence score and warn on unmatched deletes")
	fs.BoolVar(&stateMvCommands, "state-mv-commands", false, "Output legacy terraform state mv shell commands instead of HCL moved blocks")
	fs.BoolVar(&failOnUnmatchedDelete, "fail-on-unmatched-delete", false, "Exit with code 2 if any destructive resource deletion remains unmatched")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: tf-move-gen [flags] [plan.json]\n\n")
		fmt.Fprintf(stderr, "Analyzes Terraform JSON plan and generates HCL moved blocks for refactored resources.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	positional := fs.Args()
	var planReader io.Reader

	if len(positional) > 0 && positional[0] != "-" {
		f, err := os.Open(positional[0])
		if err != nil {
			fmt.Fprintf(stderr, "Error opening plan file %s: %v\n", positional[0], err)
			return 1
		}
		defer f.Close()
		planReader = f
	} else {
		// Read from stdin
		if len(positional) == 0 {
			if f, ok := stdin.(*os.File); ok {
				stat, err := f.Stat()
				if err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
					fmt.Fprintf(stderr, "Error: no plan JSON provided. Pass a file path or pipe JSON to stdin.\n\n")
					fs.Usage()
					return 1
				}
			}
		}
		planReader = stdin
	}

	plan, err := tfplan.Parse(planReader)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing plan JSON: %v\n", err)
		return 1
	}

	result, err := matcher.FindMatches(plan, matcher.Config{
		MinConfidence: minConfidence,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error analyzing resource matches: %v\n", err)
		return 1
	}

	if explain {
		for _, d := range result.UnmatchedDeletes {
			fmt.Fprintf(stderr, "[WARNING] Unmatched delete: %s\n", d.Address)
		}
	}

	genOpts := generator.Options{
		Explain:         explain,
		StateMvCommands: stateMvCommands,
		Append:          appendOut,
	}

	if outFile != "" {
		if err := generator.WriteOutput(outFile, result.Matches, genOpts); err != nil {
			fmt.Fprintf(stderr, "Error writing output to %s: %v\n", outFile, err)
			return 1
		}
	} else {
		outContent, err := generator.Generate(result.Matches, genOpts)
		if err != nil {
			fmt.Fprintf(stderr, "Error generating output: %v\n", err)
			return 1
		}
		if outContent != "" {
			fmt.Fprint(stdout, outContent)
		}
	}

	if failOnUnmatchedDelete && len(result.UnmatchedDeletes) > 0 {
		fmt.Fprintf(stderr, "Error: %d destructive resource deletion(s) remain unmatched\n", len(result.UnmatchedDeletes))
		return 2
	}

	return 0
}
