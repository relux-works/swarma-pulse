// card-v4-validate reads explicitly supplied files and emits only a verdict.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

type readOnlyRoot struct{ *os.Root }

func (r readOnlyRoot) OpenRoot(name string) (cardv4.LayoutRoot, error) {
	child, e := r.Root.OpenRoot(name)
	if e != nil {
		return nil, e
	}
	return readOnlyRoot{child}, nil
}
func readBounded(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, cardv4.MaxBytes+1))
}
func run(args []string, out, diagnostics io.Writer) (code int) {
	defer func() {
		if recover() != nil {
			fmt.Fprintln(diagnostics, "RULE internal_error: $")
			code = 2
		}
	}()
	flags := flag.NewFlagSet("card-v4-validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	version := flags.Int("version", 4, "")
	manifest := flags.String("source-manifest", "", "")
	reservations := flags.String("reservations", "", "")
	layout := flags.Bool("check-layout", false, "")
	root := flags.String("root", "", "")
	directory := flags.String("card-directory", "", "")
	if flags.Parse(args) != nil || flags.NArg() != 1 || (*version != 3 && *version != 4) {
		fmt.Fprintln(diagnostics, "RULE usage: $")
		return 2
	}
	raw, e := readBounded(flags.Arg(0))
	if e != nil {
		fmt.Fprintln(diagnostics, "RULE input_unreadable: $")
		return 2
	}
	options := cardv4.Options{Version: *version}
	if *manifest != "" {
		data, e := readBounded(*manifest)
		var m struct {
			Pins []struct {
				File     string `json:"file"`
				Revision string `json:"revision"`
				SHA256   string `json:"sha256"`
			} `json:"source_pins"`
		}
		if e != nil || len(data) > cardv4.MaxBytes || json.Unmarshal(data, &m) != nil {
			fmt.Fprintln(diagnostics, "RULE source_manifest_invalid: $")
			return 2
		}
		base, e := os.OpenRoot(filepath.Dir(*manifest))
		if e != nil {
			fmt.Fprintln(diagnostics, "RULE source_manifest_unreadable: $")
			return 2
		}
		defer base.Close()
		for _, pin := range m.Pins {
			if !filepath.IsLocal(pin.File) {
				fmt.Fprintln(diagnostics, "RULE source_path_escape: $")
				return 2
			}
			f, e := base.Open(pin.File)
			if e != nil {
				fmt.Fprintln(diagnostics, "RULE source_unreadable: $")
				return 2
			}
			b, e := io.ReadAll(io.LimitReader(f, cardv4.MaxBytes+1))
			f.Close()
			if e != nil {
				fmt.Fprintln(diagnostics, "RULE source_unreadable: $")
				return 2
			}
			kind := "v3-card"
			d, issue := cardv4.Parse(b, 3)
			if issue != nil {
				fmt.Fprintln(diagnostics, "RULE source_invalid: $")
				return 2
			}
			if d["schema"] != "product-card/v3" {
				kind = "service-descriptor"
			}
			options.Sources = append(options.Sources, cardv4.Source{Kind: kind, Revision: pin.Revision, SHA256: pin.SHA256, Bytes: b})
		}
	}
	if *reservations != "" {
		data, e := readBounded(*reservations)
		var r struct {
			Reservations []cardv4.Reservation `json:"reviewed_reservation_premises"`
		}
		if e != nil || len(data) > cardv4.MaxBytes || json.Unmarshal(data, &r) != nil {
			fmt.Fprintln(diagnostics, "RULE reservations_invalid: $")
			return 2
		}
		options.Reservations = r.Reservations
	}
	if issue := cardv4.Validate(raw, options); issue != nil {
		fmt.Fprintln(diagnostics, issue)
		return issue.Exit
	}
	if *layout {
		if *root == "" || *directory == "" {
			fmt.Fprintln(diagnostics, "RULE layout_root_required: $")
			return 2
		}
		access, e := os.OpenRoot(*root)
		if e != nil {
			fmt.Fprintln(diagnostics, "RULE layout_root_unreadable: $")
			return 2
		}
		defer access.Close()
		card, issue := cardv4.Parse(raw, *version)
		if issue != nil {
			fmt.Fprintln(diagnostics, issue)
			return issue.Exit
		}
		if issue := cardv4.CheckLayoutVersion(card, *directory, readOnlyRoot{access}, *version); issue != nil {
			fmt.Fprintln(diagnostics, issue)
			return issue.Exit
		}
	}
	fmt.Fprintln(out, "valid")
	return 0
}
