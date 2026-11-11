package pulse

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

// currentRoomRoutes consumes an already validated card and the graph's returned
// project set. It describes declarations without resolving aliases or membership.
func currentRoomRoutes(card object, projects []any) ([]any, error) {
	header := cardHeader(card)
	source := cardID(card)
	topology := obj(header["topology"])
	connection := obj(obj(card["connections"])[str(topology["connection"])])
	server := str(connection["server_name"])
	if server == "" {
		return nil, refusal("card_invalid")
	}
	entries := []any{}
	addDeclared := func(scope, key, role string, r object) {
		entries = append(entries, object{"scope": scope, "key": key, "role": role, "server_name": r["server_name"], "alias": r["alias"], "room_id": r["room_id"], "state": "declared", "membership": "unknown"})
	}
	add := func(scope, key, role, server, reference string) {
		var alias, id any
		if strings.HasPrefix(reference, "!") {
			id = reference
		} else {
			alias = reference
		}
		entries = append(entries, object{"scope": scope, "key": key, "role": role, "server_name": server, "alias": alias, "room_id": id, "state": "declared", "membership": "unknown"})
	}
	for _, role := range []string{"space", "lobby", "escalations", "upward_escalations"} {
		if header["rooms"] != nil && role != "upward_escalations" {
			continue
		}
		if reference := str(topology[role]); reference != "" {
			add("product", source, role, server, reference)
		}
	}
	for _, value := range arr(header["rooms"]) {
		r := obj(value)
		addDeclared("product", source, str(r["role"]), r)
	}
	for _, value := range arr(topology["grouped_rooms"]) {
		add("product", source, "grouped_room", server, str(value))
	}
	selected := map[string]bool{}
	for _, value := range projects {
		selected[str(obj(value)["key"])] = true
	}
	members := map[string]object{}
	for _, value := range arr(card["members"]) {
		member := obj(value)
		members[str(member["id"])] = member
	}
	for _, value := range arr(card["projects"]) {
		project := obj(value)
		key := source + "/" + str(project["id"])
		if !selected[key] {
			continue
		}
		manager := members[str(project["orchestrator"])]
		bindingKey := str(manager["binding"])
		if bindingKey == "" {
			bindingKey = str(header["default_binding"])
		}
		binding := obj(obj(card["bindings"])[bindingKey])
		connectionKey := str(binding["connection"])
		connection := obj(obj(card["connections"])[connectionKey])
		server := str(connection["server_name"])
		if manager == nil || connectionKey == "" || server == "" || obj(manager["a2a"])["connection"] != connectionKey {
			return nil, refusal("card_invalid")
		}
		if project["rooms"] != nil {
			for _, value := range arr(project["rooms"]) {
				addDeclared("project", key, "team_room", obj(value))
			}
		} else {
			add("project", key, "team_room", server, str(obj(project["channels"])["team_room"]))
		}
	}
	return orderedRoomRoutes(entries), nil
}

const pairContract = "a2a-agent-dm/v1"

// pairKey uses the S1 domain separator and exact, byte-sorted full identities.
func pairKey(sender, recipient string) string {
	ids := []string{sender, recipient}
	sort.Strings(ids)
	h := sha256.New()
	h.Write([]byte(pairContract + "\x00"))
	for _, id := range ids {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(id)))
		h.Write(size[:])
		h.Write([]byte(id))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Only declared manager classes get a descriptor. This observes no transport
// admission, mapping, membership or readiness, and cannot create a room.
func declaredDMRoutes(self *graphMember, nodes map[string]*graphMember, returned map[string]bool) ([]any, error) {
	keys := []string{}
	for key := range returned {
		if key != self.key {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	entries := []any{}
	manager := func(role any) bool { return role == "keeper" || role == "orchestrator" }
	for _, key := range keys {
		target := nodes[key]
		entry := object{"member": key, "descriptor": nil, "reason": nil}
		switch {
		case target.resolved == nil:
			entry["reason"] = "non_chat"
		case !manager(self.member["role"]):
			entry["reason"] = "caller_class_unsupported"
		case !manager(target.member["role"]):
			entry["reason"] = "target_class_unsupported"
		default:
			sender, recipient := str(self.resolved), str(target.resolved)
			if !cardv4.SupportedMXID(sender) || !cardv4.SupportedMXID(recipient) || sender == recipient {
				return nil, refusal("identity_conflict")
			}
			entry["descriptor"] = object{"contract": pairContract, "sender_mxid": sender, "recipient_mxid": recipient, "pair_key": pairKey(sender, recipient), "room_id": nil, "state": "unknown"}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Null precedes strings; identical full entries collapse, while role/scope
// distinctions survive. State and membership are constants in this projection.
func orderedRoomRoutes(entries []any) []any {
	fields := []string{"scope", "key", "role", "server_name", "alias", "room_id", "state", "membership"}
	compare := func(a, b object) int {
		for _, field := range fields {
			if a[field] == nil && b[field] != nil {
				return -1
			}
			if a[field] != nil && b[field] == nil {
				return 1
			}
			if order := strings.Compare(str(a[field]), str(b[field])); order != 0 {
				return order
			}
		}
		return 0
	}
	sort.Slice(entries, func(i, j int) bool { return compare(obj(entries[i]), obj(entries[j])) < 0 })
	result := []any{}
	for _, value := range entries {
		if len(result) == 0 || compare(obj(result[len(result)-1]), obj(value)) != 0 {
			result = append(result, value)
		}
	}
	return result
}
