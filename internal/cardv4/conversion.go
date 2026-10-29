package cardv4

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"reflect"
)

// ConversionInputs contain reviewed premises, never discovery instructions.
// ProjectLocations can confirm unchanged local placement. Changed local/remote
// dispositions refuse until a separate lossless mapping contract is supplied.
type ConversionInputs struct {
	PreservationInputs
	Reservations     []Reservation     `json:"reviewed_reservation_premises"`
	ProjectLocations map[string]Object `json:"project_locations,omitempty"`
}

// Convert returns a complete canonical JSON v4 document or no bytes on refusal.
// It performs no filesystem, process, network, registration or reservation allocation.
func Convert(raw []byte, in ConversionInputs) ([]byte, *Issue) {
	return convert(raw, in, nil)
}

// afterMember is an unexported test seam; the public path always passes nil.
func convert(raw []byte, in ConversionInputs, afterMember func(int) *Issue) ([]byte, *Issue) {
	refuse := func(e *Issue) ([]byte, *Issue) { return nil, e }
	if in.HostProject == "" || in.SourceCardRoot == "" || in.TargetCardRoot == "" || in.AdmittedRoot == "" || !revision.MatchString(in.SourceRevision) || !digest.MatchString(in.SourceSHA256) {
		return refuse(shape("conversion_inputs_missing", "$"))
	}
	for _, root := range []string{in.AdmittedRoot, in.SourceCardRoot, in.TargetCardRoot} {
		if !text(root) || filepath.Clean(root) != root {
			return refuse(shape("conversion_inputs_invalid", "$"))
		}
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != in.SourceSHA256 {
		return refuse(fail("source_digest_mismatch", "$"))
	}
	if e := Validate(raw, Options{Version: 3}); e != nil {
		return refuse(e)
	}
	source, _ := Parse(raw, 3)
	target, _ := Parse(raw, 3)
	for _, root := range []string{in.SourceCardRoot, in.TargetCardRoot} {
		if !beneath(in.AdmittedRoot, root) {
			return refuse(fail("path_root_escape", "$"))
		}
	}
	known := words("schema product external_members connections bindings profiles_root profiles projects members")
	for k := range source {
		if !known[k] {
			return refuse(fail("conversion_extension_unsupported", "$"))
		}
	}
	target["schema"] = "product-card/v4"
	product := obj(target["product"])
	product["host_project"] = in.HostProject
	projects := indexedObjects(list(source["projects"]), "id")
	host := projects[in.HostProject]
	if host == nil || filepath.Clean(filepath.Join(in.SourceCardRoot, str(host["path"]))) != filepath.Clean(in.TargetCardRoot) {
		return refuse(fail("conversion_host_project", "$.product.host_project"))
	}
	for k := range in.ProjectLocations {
		if projects[k] == nil {
			return refuse(fail("conversion_location_disposition_required", "$.projects"))
		}
	}
	externals := []any{}
	for _, v := range list(source["external_members"]) {
		mapped := in.ExternalParentMapping[str(v)]
		if mapped == nil {
			return refuse(fail("conversion_external_mapping_required", "$.external_members"))
		}
		externals = append(externals, copyObject(mapped))
	}
	target["external_members"] = externals
	connections := obj(target["connections"])
	for _, k := range sorted(in.AdditionalConnections) {
		v := in.AdditionalConnections[k]
		if old, ok := connections[k]; ok && !reflect.DeepEqual(old, v) {
			return refuse(fail("conversion_connection_preservation", "$.connections"))
		}
		connections[k] = v
	}
	bindings := obj(target["bindings"])
	for _, k := range sorted(bindings) {
		b := obj(bindings[k])
		// V3's open reader may accept a v4-looking extension here. Refuse it
		// before synthesizing identity_domain rather than overwriting its value.
		for field := range b {
			if !words("operator host connection identity_prefix")[field] {
				return refuse(fail("conversion_extension_unsupported", "$.bindings"))
			}
		}
		conn := obj(connections[str(b["connection"])])
		delete(b, "identity_prefix")
		b["identity_domain"] = conn["server_name"]
	}
	for _, k := range sorted(in.AdditionalBindings) {
		v := in.AdditionalBindings[k]
		if old, ok := bindings[k]; ok && !reflect.DeepEqual(old, v) {
			return refuse(fail("conversion_binding_preservation", "$.bindings"))
		}
		bindings[k] = v
	}
	rebase := func(value any, base, at string) (string, *Issue) {
		if !relative(value) {
			return "", fail("conversion_path_preservation", at)
		}
		absolute := filepath.Clean(filepath.Join(in.SourceCardRoot, str(value)))
		if !beneath(in.AdmittedRoot, absolute) {
			return "", fail("path_root_escape", at)
		}
		rel, e := filepath.Rel(base, absolute)
		if e != nil || !relative(rel) {
			return "", fail("conversion_path_preservation", at)
		}
		return rel, nil
	}
	profileRoot, e := rebase(target["profiles_root"], in.TargetCardRoot, "$.profiles_root")
	if e != nil {
		return refuse(e)
	}
	target["profiles_root"] = profileRoot
	for i, v := range list(target["projects"]) {
		p := obj(v)
		at := index("$.projects", i)
		path, e := rebase(p["path"], in.TargetCardRoot, at+".location")
		if e != nil {
			return refuse(e)
		}
		location := Object{"kind": "local", "path": path}
		if given := in.ProjectLocations[str(p["id"])]; given != nil {
			if given["kind"] != "local" {
				return refuse(fail("conversion_location_disposition_required", at+".location"))
			}
			if !reflect.DeepEqual(given, location) {
				return refuse(fail("conversion_path_preservation", at+".location"))
			}
		}
		task := obj(p["task_system"])
		taskPath, e := rebase(task["path"], filepath.Join(in.TargetCardRoot, path), at+".task_system.path")
		if e != nil {
			return refuse(e)
		}
		task["path"] = taskPath
		delete(p, "path")
		p["location"] = location
	}
	options := Options{Version: 4, Reservations: in.Reservations, Sources: []Source{{Kind: "v3-card", Revision: in.SourceRevision, SHA256: in.SourceSHA256, Bytes: raw}}}
	for i, v := range list(target["members"]) {
		m := obj(v)
		at := index("$.members", i)
		id := str(m["id"])
		sourceMember := obj(list(source["members"])[i])
		pin := Object{"kind": "v3-card", "revision": in.SourceRevision, "sha256": in.SourceSHA256, "member": str(product["id"]) + "/" + id}
		if !portable.MatchString(id) {
			m["logical_source_ref"] = copyObject(pin)
		}
		scope := "member"
		if m["role"] == "keeper" {
			scope = "product"
			m["project"] = in.HostProject
			mapped := in.ExternalParentMapping[str(m["parent"])]
			if mapped == nil {
				return refuse(fail("conversion_external_mapping_required", at+".parent"))
			}
			m["parent"] = copyObject(mapped)
		} else if m["role"] == "orchestrator" {
			scope = "project"
		}
		m["scope"] = scope
		if obj(m["a2a"]) != nil {
			binding := str(sourceMember["binding"])
			if binding == "" {
				binding = str(obj(source["product"])["default_binding"])
			}
			m["binding"] = binding
			name, _, e := resolveSource(pin, obj(source["product"]), id, options, true)
			if e != nil {
				return refuse(relocated(e, at+".identity.source_ref"))
			}
			m["identity"] = Object{"scheme": "legacy-exact", "mxid": name, "source_ref": pin}
		} else {
			choice := in.ProspectiveNonchatChoices[str(pin["member"])]
			if choice == nil {
				return refuse(fail("conversion_identity_preservation", at+".identity"))
			}
			m["binding"], m["identity"] = choice["binding"], choice["identity"]
		}
		for _, key := range []string{"clone", "git_dir"} {
			if value, ok := m[key]; ok {
				path, e := rebase(value, in.TargetCardRoot, at+"."+key)
				if e != nil {
					return refuse(e)
				}
				m[key] = path
			}
		}
		if afterMember != nil {
			if e := afterMember(i + 1); e != nil {
				return refuse(e)
			}
		}
	}
	output, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return refuse(shape("conversion_encoding", "$"))
	}
	output = append(output, '\n')
	if e := Validate(output, options); e != nil {
		return refuse(e)
	}
	// Reparse the serialized result: preservation covers exactly the emitted bytes.
	emitted, e := Parse(output, 4)
	if e != nil {
		return refuse(e)
	}
	if e := VerifyConversionPreservation(source, emitted, in.PreservationInputs); e != nil {
		return refuse(e)
	}
	return output, nil
}
