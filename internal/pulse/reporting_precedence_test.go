package pulse

import (
	"fmt"
	"testing"
)

func reportingChain(hops int) map[string]*graphMember {
	nodes := map[string]*graphMember{}
	for i := 0; i <= hops; i++ {
		key := fmt.Sprintf("%03d", i)
		n := &graphMember{key: key}
		if i > 0 {
			n.parent = fmt.Sprintf("%03d", i-1)
		}
		nodes[key] = n
	}
	return nodes
}

// A reporting limit on one walk must not mask a conflict found by another
// bounded walk. Keep the valid boundary and the adjacent refusal beside it.
func TestReportingErrorPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		hops   int
		change func(map[string]*graphMember)
		rule   string
	}{
		{"valid-64", 64, nil, ""},
		{"limit-65", 65, nil, "input_limit"},
		{"tail-into-cycle", 64, func(n map[string]*graphMember) { n["000"].parent = "001" }, "relation_conflict"},
		{"limit-and-disjoint-cycle", 65, func(n map[string]*graphMember) {
			n["cycle"] = &graphMember{key: "cycle", parent: "cycle"}
		}, "relation_conflict"},
		{"limit-and-dangling-edge", 65, func(n map[string]*graphMember) {
			n["dangling"] = &graphMember{key: "dangling", parent: "missing"}
		}, "relation_conflict"},
		{"cycle-64", 63, func(n map[string]*graphMember) { n["000"].parent = "063" }, "relation_conflict"},
		// A cycle that no 64-hop walk can close still exhausts the budget.
		{"cycle-65", 64, func(n map[string]*graphMember) { n["000"].parent = "064" }, "input_limit"},
		{"external-64", 63, func(n map[string]*graphMember) {
			n["000"].parent = object{"source": "other", "external": "keeper"}
		}, ""},
		{"external-65", 64, func(n map[string]*graphMember) {
			n["000"].parent = object{"source": "other", "external": "keeper"}
		}, "input_limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes := reportingChain(tc.hops)
			if tc.change != nil {
				tc.change(nodes)
			}
			err := checkReportingGraph(nodes)
			if tc.rule == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				wantRule(t, err, tc.rule)
			}
		})
	}
}
