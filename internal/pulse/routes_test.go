package pulse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Existing v3/v4 room declarations retain opaque IDs, foreign aliases and
// independent routing servers. Only returned projects are included, and only
// identical complete entries deduplicate; different roles sharing a room survive.
func TestCurrentRoomRoutes(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			c := fixture(t)
			if version == "v4" {
				c = v4Fixture(t)
			}
			topology := obj(obj(c["product"])["topology"])
			topology["space"] = "!AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
			topology["lobby"] = "#shared:foreign.example"
			topology["escalations"] = "!legacyRoom:foreign.example"
			topology["grouped_rooms"] = []any{"#shared:foreign.example", "!groupedRoom:foreign.example", "#shared:foreign.example"}
			topology["upward_escalations"] = "#shared:foreign.example"
			raw := encode(t, c)
			if _, err := decodeQueryCard(raw, "sha256:"+digest(raw), false); err != nil {
				t.Fatal("invalid positive premise", err)
			}
			projects := []any{object{"key": "aurora/beacon"}}
			got, err := currentRoomRoutes(c, projects)
			if err != nil {
				t.Fatal(err)
			}
			entry := func(scope, key, role string, alias, id any) object {
				return object{"scope": scope, "key": key, "role": role, "server_name": "relux.works", "alias": alias, "room_id": id, "state": "declared", "membership": "unknown"}
			}
			want := []any{
				entry("product", "aurora", "escalations", nil, "!legacyRoom:foreign.example"),
				entry("product", "aurora", "grouped_room", nil, "!groupedRoom:foreign.example"),
				entry("product", "aurora", "grouped_room", "#shared:foreign.example", nil),
				entry("product", "aurora", "lobby", "#shared:foreign.example", nil),
				entry("product", "aurora", "space", nil, "!AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
				entry("product", "aurora", "upward_escalations", "#shared:foreign.example", nil),
				entry("project", "aurora/beacon", "team_room", "#beacon:relux.works", nil),
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("want %#v, got %#v", want, got)
			}
		})
	}
}

func routesView(t *testing.T, c object, agent string) object {
	t.Helper()
	raw := encode(t, c)
	b, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", agent, fixedTime, "routes")
	if err != nil {
		t.Fatal(err)
	}
	var v object
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func addDeclaredRooms(c object) {
	obj(c["product"])["rooms"] = []any{
		object{"role": "space", "server_name": "relux.works", "alias": "#space:foreign.example", "room_id": "!AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
		object{"role": "lobby", "server_name": "relux.works", "alias": "#lobby:foreign.example"},
		object{"role": "escalations", "server_name": "relux.works", "room_id": "!escalationsRoom:foreign.example"},
	}
	for i, value := range arr(c["projects"]) {
		obj(value)["rooms"] = []any{object{"role": "project_room", "server_name": "relux.works", "alias": fmt.Sprintf("#project-%d:foreign.example", i), "room_id": fmt.Sprintf("!projectRoom%02d:foreign.example", i)}}
	}
}

// Routes reuse every hierarchy field and relationship. DM entries deduplicate
// logical members, exclude self, and never claim transport or membership state.
func TestRoutesProjection(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			c := fixture(t)
			if version == "v4" {
				c = v4Fixture(t)
			}
			v := routesView(t, c, "station_aurora_keeper")
			if len(v) != 19 || v["schema"] != "pulse-view/v4" || !reflect.DeepEqual(v["query"], object{"view": "routes"}) {
				t.Fatal(v)
			}
			baseline := hierarchyView(t, c, "station_aurora_keeper")
			for k, value := range baseline {
				if k != "schema" && k != "query" && !reflect.DeepEqual(value, v[k]) {
					t.Fatal("hierarchy changed", k)
				}
			}
			routes := arr(v["dm_routes"])
			if len(routes) != 3 {
				t.Fatal(routes)
			}
			for i, key := range []string{"aurora/beacon_orch", "aurora/storage_orch", "aurora/vault_dev"} {
				r := obj(routes[i])
				if len(r) != 3 || r["member"] != key {
					t.Fatal(r)
				}
				if i == 2 {
					if r["descriptor"] != nil || r["reason"] != "non_chat" {
						t.Fatal(r)
					}
					continue
				}
				d := obj(r["descriptor"])
				if len(d) != 6 || r["reason"] != nil || d["contract"] != pairContract || d["sender_mxid"] != "@station_aurora_keeper:relux.works" || d["recipient_mxid"] != "@station_"+strings.TrimPrefix(key, "aurora/")+":relux.works" || d["room_id"] != nil || d["state"] != "unknown" {
					t.Fatal(r)
				}
			}
			forward := obj(obj(routes[0])["descriptor"])
			reverseView := routesView(t, c, "station_beacon_orch")
			reverse := obj(obj(arr(reverseView["dm_routes"])[0])["descriptor"])
			if forward["pair_key"] != reverse["pair_key"] || forward["sender_mxid"] != reverse["recipient_mxid"] || forward["recipient_mxid"] != reverse["sender_mxid"] {
				t.Fatal(forward, reverse)
			}
			for _, r := range arr(v["room_routes"]) {
				if len(obj(r)) != 8 || obj(r)["membership"] != "unknown" || obj(r)["state"] != "declared" {
					t.Fatal(r)
				}
			}
		})
	}
}

// Fixed S1 vectors independently encode domain+NUL and uint32 lengths. Reverse
// directions share a key; equal localparts on different servers remain distinct.
func TestPairKeyVectors(t *testing.T) {
	for _, tc := range []struct{ a, b, key string }{
		{"@keeper:alpha.example", "@orch:beta.example", "01008d925500d010d98d8a3fec2ceb6721d87e1214650050800ca4e616090e6f"},
		{"@same:alpha.example", "@same:beta.example", "a04bad9cf736e9dd8e2e8283bffd1e2b7fa05b12d0a9a0981dfa259d21962c98"},
	} {
		if pairKey(tc.a, tc.b) != tc.key || pairKey(tc.b, tc.a) != tc.key {
			t.Fatal(tc)
		}
		if pairKey(tc.a, tc.b+"x") == tc.key {
			t.Fatal("identity collapsed")
		}
	}
}

// A chat-enabled developer still has no manager-class descriptor. Non-chat
// takes precedence; an impossible self-identity refuses with no partial array.
func TestDMUnsupportedAndSelfPair(t *testing.T) {
	c := fixture(t)
	dev := obj(arr(c["members"])[3])
	dev["result_route"], dev["chat_admission"] = "a2a", "fixture-admission"
	dev["a2a"] = obj(arr(c["members"])[2])["a2a"]
	keeper := routesView(t, c, "station_aurora_keeper")
	r := obj(arr(keeper["dm_routes"])[2])
	if r["descriptor"] != nil || r["reason"] != "target_class_unsupported" {
		t.Fatal(r)
	}
	worker := routesView(t, c, "station_vault_dev")
	for _, value := range arr(worker["dm_routes"]) {
		if obj(value)["descriptor"] != nil || obj(value)["reason"] != "caller_class_unsupported" {
			t.Fatal(value)
		}
	}
	self := &graphMember{key: "p/self", member: object{"role": "keeper"}, resolved: "@same:alpha.example"}
	target := &graphMember{key: "p/other", member: object{"role": "orchestrator"}, resolved: self.resolved}
	got, err := declaredDMRoutes(self, map[string]*graphMember{target.key: target}, map[string]bool{target.key: true})
	wantRule(t, err, "identity_conflict")
	if got != nil {
		t.Fatal("partial self pair", got)
	}
	target.resolved = nil
	self.member["role"] = "developer"
	got, err = declaredDMRoutes(self, map[string]*graphMember{target.key: target}, map[string]bool{target.key: true})
	if err != nil || obj(got[0])["reason"] != "non_chat" {
		t.Fatal(got, err)
	}
}

// Explicit metadata replaces the corresponding scalar roles, retains both
// references, and stays private in existing views except truthful source pins.
func TestRoutesAmendedRooms(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			c := fixture(t)
			if version == "v4" {
				c = v4Fixture(t)
			}
			before := hierarchyView(t, c, "station_aurora_keeper")
			addDeclaredRooms(c)
			v := routesView(t, c, "station_aurora_keeper")
			foundSpace := false
			for _, value := range arr(v["room_routes"]) {
				r := obj(value)
				if r["role"] == "space" {
					if r["alias"] != "#space:foreign.example" || r["room_id"] != "!AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" || r["server_name"] != "relux.works" {
						t.Fatal(r)
					}
					foundSpace = true
				}
				if r["role"] == "team_room" && (r["alias"] == nil || r["room_id"] == nil) {
					t.Fatal(r)
				}
			}
			if !foundSpace {
				t.Fatal("missing space")
			}
			after := hierarchyView(t, c, "station_aurora_keeper")
			delete(before, "sources")
			delete(after, "sources")
			if !reflect.DeepEqual(before, after) {
				t.Fatal("metadata leaked")
			}
			for _, view := range []string{"", "environment", "hierarchy"} {
				raw := encode(t, c)
				output, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_aurora_keeper", fixedTime, view)
				if version == "v4" && view == "environment" {
					wantRule(t, err, "unsupported_schema")
					if output != nil {
						t.Fatal("partial unsupported environment")
					}
					continue
				}
				if err != nil || bytes.Contains(output, []byte("foreign.example")) {
					t.Fatal("room metadata disclosed", view, err)
				}
			}
			delete(obj(c["product"]), "rooms")
			raw := encode(t, c)
			output, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_aurora_keeper", fixedTime, "routes")
			wantRule(t, err, "card_invalid")
			if output != nil {
				t.Fatal("partial invalid set")
			}
		})
	}
}

// CLI routes is explicit, refuses extra sources before reads, wrong pins and
// unsafe paths with empty stdout, and leaves the admitted input unchanged.
func TestRoutesCLI(t *testing.T) {
	raw := validBytes(t, fixture(t))
	dir := ownDirectory(t)
	path := writeFixture(t, dir, raw)
	before, _ := os.Stat(path)
	args := []string{"me", "--local-unsigned", "--view", "routes", "--card", path, "--revision", "sha256:" + digest(raw), "--format", "json"}
	t.Setenv("A2A_AGENT", "station_aurora_keeper")
	var out, diagnostics bytes.Buffer
	if code := Run(args, &out, &diagnostics); code != 0 || diagnostics.Len() != 0 || !json.Valid(out.Bytes()) {
		t.Fatal(code, diagnostics.String())
	}
	for _, flag := range []string{"--sources", "--sources-revision", "--root", "--view"} {
		bad := append(append([]string{}, args...), flag, "/unread/fixture")
		out.Reset()
		diagnostics.Reset()
		if code := Run(bad, &out, &diagnostics); code != 2 || out.Len() != 0 || diagnostics.String() != "RULE usage: $\n" {
			t.Fatal(flag, code, diagnostics.String())
		}
	}
	s, err := parseSelectors(args)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"source_digest", "input_trust"} {
		bad := s
		if rule == "source_digest" {
			bad.revision = "sha256:" + strings.Repeat("0", 64)
		} else {
			bad.card = filepath.Join(dir, "link")
			if err := os.Symlink(path, bad.card); err != nil {
				t.Fatal(err)
			}
		}
		out.Reset()
		diagnostics.Reset()
		if code := runSelected(bad, os.Getuid(), os.Geteuid(), "station_aurora_keeper", fixedTime, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.String() != "RULE "+rule+": $\n" {
			t.Fatal(rule, code, diagnostics.String())
		}
	}
	after, _ := os.Stat(path)
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(raw, actual) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("input mutated", err)
	}
}

// These exact bytes were captured from clean pre-routes head102097a2, using
// fixed synthetic inputs/time. The separate discovery base fixture remains in use.
func TestBeforeRoutesByteCompatibility(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		c := fixture(t)
		if version == "v4" {
			c = v4Fixture(t)
		}
		raw := encode(t, c)
		for _, view := range []string{"", "environment", "hierarchy"} {
			if version == "v4" && view == "environment" {
				continue
			}
			name := view
			if name == "" {
				name = "self"
			}
			expected, err := os.ReadFile(filepath.Join("testdata", "before-routes-"+version+"-"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			actual, err := projectView(raw, "sha256:"+digest(raw), "/fixture/card", "station_aurora_keeper", fixedTime, view)
			if err != nil || !bytes.Equal(actual, expected) {
				t.Fatalf("%s/%s changed from base102097a2: %v", version, name, err)
			}
		}
	}
}

// Project routing follows the manager's explicit binding rather than the
// caller/product server. An a2a/binding conflict refuses without partial routes.
func TestCurrentRoomManagerBinding(t *testing.T) {
	c := fixture(t)
	obj(c["connections"])["other"] = object{"homeserver": "https://other.example", "server_name": "other.example"}
	obj(c["bindings"])["other"] = object{"operator": "fixture", "host": "other", "identity_prefix": "other_", "connection": "other"}
	m := obj(arr(c["members"])[1])
	m["binding"] = "other"
	obj(m["a2a"])["connection"] = "other"
	_ = validBytes(t, c)
	projects := []any{object{"key": "aurora/beacon"}}
	got, err := currentRoomRoutes(c, projects)
	if err != nil {
		t.Fatal(err)
	}
	project := obj(got[len(got)-1])
	if project["server_name"] != "other.example" || project["alias"] != "#beacon:relux.works" {
		t.Fatal(project)
	}
	obj(m["a2a"])["connection"] = "legacy-relux"
	got, err = currentRoomRoutes(c, projects)
	wantRule(t, err, "card_invalid")
	if got != nil {
		t.Fatal("partial invalid routes", got)
	}
}
