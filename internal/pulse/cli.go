package pulse

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type selectors struct{ card, revision, view, sources, sourcesRevision, root string }

func parseSelectors(args []string) (selectors, error) {
	var s selectors
	if len(args) == 0 || args[0] != "me" {
		return s, refusal("usage")
	}
	values := map[string]string{}
	local := false
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--local-unsigned" {
			if local {
				return s, refusal("usage")
			}
			local = true
			continue
		}
		if !strings.HasPrefix(arg, "--") {
			return s, refusal("usage")
		}
		key, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if key != "card" && key != "revision" && key != "format" && key != "view" && key != "sources" && key != "sources-revision" && key != "root" {
			return s, refusal("usage")
		}
		if _, duplicate := values[key]; duplicate {
			return s, refusal("usage")
		}
		if !hasValue {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return s, refusal("usage")
			}
			value = args[i]
		}
		if value == "" {
			return s, refusal("usage")
		}
		values[key] = value
	}
	if !local || values["card"] == "" || values["format"] != "json" {
		return s, refusal("usage")
	}
	if view, present := values["view"]; present && view != "environment" && view != "discovery" && view != "hierarchy" {
		return s, refusal("usage")
	}
	if !validRevision(values["revision"]) {
		return s, refusal("usage")
	}
	if values["view"] == "discovery" {
		if values["sources"] == "" || !validRevision(values["sources-revision"]) || !cleanAbsolute(values["root"]) {
			return s, refusal("usage")
		}
	} else if values["sources"] != "" || values["sources-revision"] != "" || values["root"] != "" {
		return s, refusal("usage")
	}
	return selectors{card: values["card"], revision: values["revision"], view: values["view"], sources: values["sources"], sourcesRevision: values["sources-revision"], root: values["root"]}, nil
}

// Run requires affirmative unsigned mode before reading OS context or any file.
// A2A_AGENT is an unverified local claim, not a trust anchor or cryptographic proof.
func Run(args []string, out, diagnostics io.Writer) int {
	s, err := parseSelectors(args)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 2
	}
	return runSelected(s, os.Getuid(), os.Geteuid(), os.Getenv("A2A_AGENT"), time.Now(), out, diagnostics)
}
func runSelected(s selectors, uid, euid int, agent string, now time.Time, out, diagnostics io.Writer) int {
	if uid != euid {
		fmt.Fprintln(diagnostics, refusal("os_principal"))
		return 1
	}
	if agent == "" {
		fmt.Fprintln(diagnostics, refusal("caller_context"))
		return 1
	}
	var result []byte
	var err error
	if s.view == "discovery" {
		var cards []*querySource
		var manifest object
		cards, manifest, err = loadDiscovery(s, uid)
		if err == nil {
			result, err = projectDiscovery(cards, manifest, agent, now)
		}
	} else {
		var raw []byte
		raw, err = readCard(s.card, uid)
		if err == nil {
			result, err = projectView(raw, s.revision, filepath.Dir(s.card), agent, now, s.view)
		}
	}
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	if _, err = out.Write(result); err != nil {
		fmt.Fprintln(diagnostics, refusal("output_write"))
		return 1
	}
	return 0
}
