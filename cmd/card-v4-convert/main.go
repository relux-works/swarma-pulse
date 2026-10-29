// card-v4-convert publishes one explicitly requested, validated conversion.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/relux-works/swarma-pulse/internal/cardpublish"
	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func readInput(path string) ([]byte, fs.FileInfo, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, nil, e
	}
	return readOpened(f)
}
func readOpened(f *os.File) ([]byte, fs.FileInfo, error) {
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return nil, nil, fs.ErrInvalid
	}
	raw, e := io.ReadAll(io.LimitReader(f, cardv4.MaxBytes+1))
	if e != nil || len(raw) > cardv4.MaxBytes {
		return nil, nil, fs.ErrInvalid
	}
	return raw, info, nil
}
func run(args []string, out, diagnostics io.Writer) int {
	refuse := func(rule string, exit int) int { fmt.Fprintf(diagnostics, "RULE %s: $\n", rule); return exit }
	flags := flag.NewFlagSet("card-v4-convert", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	source := flags.String("source", "", "")
	inputs := flags.String("inputs", "", "")
	destination := flags.String("destination", "", "")
	rootPath := flags.String("root", "", "")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *source == "" || *inputs == "" || *destination == "" || *rootPath == "" {
		return refuse("usage", 2)
	}
	premises, _, e := readInput(*inputs)
	if e != nil {
		return refuse("conversion_inputs_unreadable", 2)
	}
	var in cardv4.ConversionInputs
	if json.Unmarshal(premises, &in) != nil {
		return refuse("conversion_inputs_invalid", 2)
	}
	// Roots in premises are explicit lexical coordinates. The output-root flag
	// supplies their physical admitted anchor; neither base is guessed from a key.
	admitted := filepath.Clean(in.AdmittedRoot)
	relative, e := filepath.Rel(admitted, in.TargetCardRoot)
	if in.AdmittedRoot == "" || e != nil || !filepath.IsLocal(relative) {
		return refuse("conversion_inputs_missing", 2)
	}
	if filepath.Clean(filepath.Dir(*destination)) != relative {
		return refuse("conversion_destination_base_mismatch", 1)
	}
	sourceBase, e := filepath.Rel(admitted, in.SourceCardRoot)
	if e != nil || !filepath.IsLocal(sourceBase) || !filepath.IsLocal(*source) || !filepath.IsLocal(*destination) || filepath.Clean(filepath.Dir(*source)) != sourceBase {
		return refuse("conversion_source_base_mismatch", 1)
	}
	root, e := os.OpenRoot(*rootPath)
	if e != nil {
		return refuse("publication_root_unreadable", 2)
	}
	defer root.Close()
	f, e := root.Open(filepath.Clean(*source))
	if e != nil {
		return refuse("source_unreadable", 2)
	}
	raw, info, e := readOpened(f)
	if e != nil {
		return refuse("source_unreadable", 2)
	}
	complete, issue := cardv4.Convert(raw, in)
	if issue != nil {
		fmt.Fprintf(diagnostics, "RULE %s: %s\n", issue.Rule, issue.Location)
		return issue.Exit
	}
	result, pubIssue := cardpublish.Publish(root, *destination, info, complete)
	if pubIssue != nil {
		if result.Published {
			return refuse("publication_complete_durability_unconfirmed", 3)
		}
		return refuse(pubIssue.Rule, 1)
	}
	fmt.Fprintln(out, "converted")
	return 0
}
