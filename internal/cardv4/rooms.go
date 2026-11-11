package cardv4

import "strings"

// Entries are closed even in v3. References are preserved, never resolved.
func roomCollection(version int, roles string) validator {
	server := dns
	if version == 3 {
		server = func(v any) bool { return legacyServer.MatchString(str(v)) }
	}
	reference := func(v any, alias bool) bool {
		parts := strings.Split(str(v), ":")
		if alias {
			return len(parts) == 2 && aliasLocal.MatchString(parts[0]) && server(parts[1])
		}
		return len(parts) <= 2 && roomLocal.MatchString(parts[0]) && (len(parts) == 1 || server(parts[1]))
	}
	entry := object(map[string]validator{
		"role": enumeration(roles), "server_name": scalar(server, "room_declaration"),
	}, map[string]validator{
		"alias":   scalar(func(v any) bool { return reference(v, true) }, "room_declaration"),
		"room_id": scalar(func(v any) bool { return reference(v, false) }, "room_declaration"),
	}, true)
	validate := array(func(v any, at string) *Issue {
		if e := entry(v, at); e != nil {
			return e
		}
		m := obj(v)
		if _, alias := m["alias"]; !alias {
			if _, id := m["room_id"]; !id {
				return fail("room_declaration", at)
			}
		}
		return nil
	}, true)
	return validate
}

// Opting in requires all scopes. Routing follows the selected connection,
// independently of alias hosts and opaque or legacy room-ID suffixes.
func validateRoomDeclarations(c Object) *Issue {
	product := obj(c["product"])
	_, enabled := product["rooms"]
	for _, p := range list(c["projects"]) {
		_, present := obj(p)["rooms"]
		enabled = enabled || present
	}
	if !enabled {
		return nil
	}
	bindings, connections := obj(c["bindings"]), obj(c["connections"])
	ids := map[string]string{}
	check := func(owner Object, at, server string, count int) *Issue {
		rooms := list(owner["rooms"])
		if len(rooms) != count {
			return fail("room_declaration", at)
		}
		roles := map[string]bool{}
		for i, value := range rooms {
			r := obj(value)
			location := index(at, i)
			role := str(r["role"])
			if roles[role] || r["server_name"] != server {
				return fail("room_declaration", location)
			}
			roles[role] = true
			if id := str(r["room_id"]); id != "" {
				if previous, exists := ids[id]; exists && previous != at {
					return fail("room_declaration", location+".room_id")
				}
				ids[id] = at
			}
		}
		return nil
	}
	server := str(obj(connections[str(obj(product["topology"])["connection"])])["server_name"])
	if e := check(product, "$.product.rooms", server, 3); e != nil {
		return e
	}
	members := map[string]Object{}
	for _, value := range list(c["members"]) {
		m := obj(value)
		members[str(m["id"])] = m
	}
	for i, value := range list(c["projects"]) {
		p := obj(value)
		manager := members[str(p["orchestrator"])]
		key := str(manager["binding"])
		if key == "" {
			key = str(product["default_binding"])
		}
		server := str(obj(connections[str(obj(bindings[key])["connection"])])["server_name"])
		if e := check(p, index("$.projects", i)+".rooms", server, 1); e != nil {
			return e
		}
	}
	return nil
}
