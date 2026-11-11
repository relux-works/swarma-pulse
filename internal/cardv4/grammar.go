package cardv4

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	portable      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	slug          = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	revision      = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,200}$`)
	digest        = regexp.MustCompile(`^[a-f0-9]{64}$`)
	commit        = regexp.MustCompile(`^[a-f0-9]{7,40}$`)
	semver        = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	legacyVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	legacyServer  = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,252}$`)
	legacyHTTPS   = regexp.MustCompile(`^https://[A-Za-z0-9.-]+(:[0-9]+)?/?$`)
	domainLabel   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	aliasLocal    = regexp.MustCompile(`^#[a-z0-9][a-z0-9._-]{0,63}$`)
	roomLocal     = regexp.MustCompile(`^![A-Za-z0-9_.~-]{8,255}$`)
	mxidLocal     = regexp.MustCompile(`^@[a-z0-9._=/-]{1,255}$`)
	credentialKey = regexp.MustCompile(`(?i)(token|password|passwd|secret|private_key|api_key|credential)`)
	secretPattern = regexp.MustCompile(`(gh[pousr]_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9-]{20,}|xox[abpr]-[A-Za-z0-9-]{10,}|-----BEGIN [A-Z ]*PRIVATE KEY-----|sk-ant-[A-Za-z0-9-]{20,}|syt_[A-Za-z0-9_]{20,})`)
)

func text(v any) bool {
	s, ok := v.(string)
	return ok && s != "" && !strings.ContainsFunc(s, unicode.IsControl)
}
func logical(v any) bool { s, ok := v.(string); return ok && s != "" }
func dns(v any) bool {
	s := str(v)
	if len(s) > 253 || !strings.Contains(s, ".") {
		return false
	}
	for _, p := range strings.Split(s, ".") {
		if !domainLabel.MatchString(p) {
			return false
		}
	}
	return true
}
func room(v any) bool {
	s := str(v)
	parts := strings.Split(s, ":")
	return len(parts) <= 2 && (aliasLocal.MatchString(parts[0]) || roomLocal.MatchString(parts[0])) && (len(parts) == 1 || dns(parts[1]))
}
func mxid(v any) bool {
	p := strings.Split(str(v), ":")
	return len(p) == 2 && mxidLocal.MatchString(p[0]) && dns(p[1])
}

// SupportedMXID checks the strict full identity form used for outgoing route
// descriptors. It does not normalize a value or grant transport admission.
func SupportedMXID(value string) bool { return mxid(value) }
func origin(v any) bool {
	u, e := url.Parse(str(v))
	return e == nil && !strings.Contains(str(v), "#") && u.Scheme == "https" && u.Host != "" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && (u.Path == "" || u.Path == "/") && dns(u.Hostname()) && u.Opaque == ""
}
func relative(v any) bool {
	s := str(v)
	return text(v) && !strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "~")
}
func repo(v any) bool {
	if !text(v) || strings.ContainsFunc(str(v), unicode.IsSpace) {
		return false
	}
	p := strings.Split(str(v), "/")
	if len(p) != 3 || !dns(p[0]) {
		return false
	}
	for _, s := range p {
		if s == "" || strings.ContainsAny(s, " \t\r\n@:?#\\") {
			return false
		}
	}
	return true
}
func compatible(selected, requirement string) bool {
	return compatibleVersion(selected, requirement, false)
}
func validRange(requirement string, legacy bool) bool {
	parts := strings.Split(requirement, " ")
	valid := semver.MatchString
	if legacy {
		parts = strings.Fields(requirement)
		valid = legacyVersion.MatchString
	}
	if len(parts) < 1 || len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		op := ""
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(p, candidate) {
				op = candidate
				break
			}
		}
		if op == "" || !valid(strings.TrimPrefix(p, op)) {
			return false
		}
	}
	return true
}
func compatibleVersion(selected, requirement string, legacy bool) bool {
	valid := semver.MatchString
	if legacy {
		valid = legacyVersion.MatchString
	}
	if !valid(selected) {
		return false
	}
	a := strings.Split(selected, ".")
	parts := strings.Split(requirement, " ")
	if legacy {
		parts = strings.Fields(requirement)
	}
	if len(parts) < 1 || len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		op := ""
		for _, v := range []string{">=", "<=", "=", ">", "<"} {
			if strings.HasPrefix(p, v) {
				op = v
				break
			}
		}
		if op == "" {
			return false
		}
		other := strings.TrimPrefix(p, op)
		if !valid(other) {
			return false
		}
		b := strings.Split(other, ".")
		cmp := 0
		for i := range a {
			x, y := strings.TrimLeft(a[i], "0"), strings.TrimLeft(b[i], "0")
			if len(x) < len(y) || len(x) == len(y) && x < y {
				cmp = -1
				break
			}
			if len(x) > len(y) || len(x) == len(y) && x > y {
				cmp = 1
				break
			}
		}
		if !(op == "=" && cmp == 0 || op == ">" && cmp > 0 || op == "<" && cmp < 0 || op == ">=" && cmp >= 0 || op == "<=" && cmp <= 0) {
			return false
		}
	}
	return true
}

var roles = words("keeper orchestrator developer tester architect reviewer researcher service")
var kinds = words("proposal question answer decision ack notice offer request accept decline handover subscribe unsubscribe owner-needed system")
var roleCodes = map[string]string{"keeper": "keeper", "orchestrator": "orch", "developer": "dev", "tester": "tester", "architect": "arch", "reviewer": "reviewer", "researcher": "researcher", "service": "service"}
