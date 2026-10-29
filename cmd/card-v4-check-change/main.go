// card-v4-check-change verifies explicitly supplied desired-change snapshots.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// The admitted snapshot tree must stay under its custodian's exclusive write
// custody during a check. Rooted reads constrain paths; they do not freeze files.
func readWithin(root *os.Root, name string) ([]byte, error) {
	if !filepath.IsLocal(name) {
		return nil, fs.ErrInvalid
	}
	name = filepath.Clean(name)
	info, e := root.Stat(name)
	if e != nil || !info.Mode().IsRegular() {
		return nil, fs.ErrInvalid
	}
	f, e := root.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e = f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return nil, fs.ErrInvalid
	}
	raw, e := io.ReadAll(io.LimitReader(f, cardv4.MaxBytes+1))
	if e != nil || len(raw) > cardv4.MaxBytes {
		return nil, fs.ErrInvalid
	}
	return raw, nil
}

// Metadata uses the same bounded, duplicate-free mapping parser. Fixture
// observation fields are not authority; only the named list is decoded.
func readList(root *os.Root, file, key string, destination any) error {
	raw, e := readWithin(root, file)
	if e != nil {
		return e
	}
	doc, issue := cardv4.Parse(raw, 4)
	if issue != nil {
		return fs.ErrInvalid
	}
	if _, ok := doc[key].([]any); !ok {
		return fs.ErrInvalid
	}
	raw, e = json.Marshal(doc[key])
	if e != nil {
		return e
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func readOptions(root *os.Root, manifest, reservations string) (cardv4.Options, *cardv4.Issue) {
	o := cardv4.Options{Version: 4}
	refuse := func(rule string, exit int) (cardv4.Options, *cardv4.Issue) {
		return o, &cardv4.Issue{Rule: rule, Location: "$", Exit: exit}
	}
	var pins []struct {
		File     string `json:"file"`
		Revision string `json:"revision"`
		SHA256   string `json:"sha256"`
	}
	if readList(root, manifest, "source_pins", &pins) != nil {
		return refuse("source_manifest_invalid", 2)
	}
	// Each source stays inside the pinned manifest's directory as well as the
	// admitted root. Relative path spelling never selects another authority.
	if !filepath.IsLocal(manifest) {
		return refuse("source_manifest_invalid", 2)
	}
	base, e := root.OpenRoot(filepath.Clean(filepath.Dir(manifest)))
	if e != nil {
		return refuse("source_manifest_invalid", 2)
	}
	defer base.Close()
	for _, pin := range pins {
		if pin.Revision == "" || pin.SHA256 == "" || !filepath.IsLocal(pin.File) {
			return refuse("source_manifest_invalid", 2)
		}
		raw, e := readWithin(base, pin.File)
		if e != nil {
			return refuse("source_unreadable", 2)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != pin.SHA256 {
			return refuse("source_digest_mismatch", 1)
		}
		doc, issue := cardv4.Parse(raw, 3)
		if issue != nil {
			return refuse("source_invalid", 2)
		}
		kind := "service-descriptor"
		if doc["schema"] == "product-card/v3" {
			kind = "v3-card"
		}
		o.Sources = append(o.Sources, cardv4.Source{Kind: kind, Revision: pin.Revision, SHA256: pin.SHA256, Bytes: raw})
	}
	if readList(root, reservations, "reviewed_reservation_premises", &o.Reservations) != nil {
		return refuse("reservations_invalid", 2)
	}
	return o, nil
}

func run(args []string, out, diagnostics io.Writer) int {
	refuse := func(issue *cardv4.Issue) int { fmt.Fprintln(diagnostics, issue); return issue.Exit }
	flags := flag.NewFlagSet("card-v4-check-change", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	rootPath := flags.String("root", "", "")
	pilot := flags.String("pilot", "", "")
	manifest := flags.String("source-manifest", "", "")
	reservations := flags.String("reservations", "", "")
	var names, hashes [3]*string
	for i, name := range []string{"prior", "candidate", "rollback"} {
		names[i] = flags.String(name, "", "")
		hashes[i] = flags.String(name+"-sha256", "", "")
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || *rootPath == "" || *pilot == "" || *manifest == "" || *reservations == "" {
		return refuse(&cardv4.Issue{Rule: "usage", Location: "$", Exit: 2})
	}
	for i := range names {
		if *names[i] == "" || *hashes[i] == "" {
			return refuse(&cardv4.Issue{Rule: "usage", Location: "$", Exit: 2})
		}
	}
	root, e := os.OpenRoot(*rootPath)
	if e != nil {
		return refuse(&cardv4.Issue{Rule: "input_root_unreadable", Location: "$", Exit: 2})
	}
	defer root.Close()
	var snapshots [3]cardv4.Snapshot
	for i := range names {
		raw, e := readWithin(root, *names[i])
		if e != nil {
			return refuse(&cardv4.Issue{Rule: "snapshot_unreadable", Location: fmt.Sprintf("$[%d]", i), Exit: 2})
		}
		snapshots[i] = cardv4.Snapshot{Bytes: raw, SHA256: *hashes[i]}
	}
	o, issue := readOptions(root, *manifest, *reservations)
	if issue != nil {
		return refuse(issue)
	}
	if issue := cardv4.CheckChange(snapshots[0], snapshots[1], snapshots[2], *pilot, o); issue != nil {
		return refuse(issue)
	}
	fmt.Fprintln(out, "change-valid")
	return 0
}
