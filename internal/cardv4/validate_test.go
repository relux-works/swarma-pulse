package cardv4

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const fixtures = "../../docs/fixtures/v4/full"

func fixtureBytes(t *testing.T, file string) []byte {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(fixtures, file))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func fixture(t *testing.T, file string) Object {
	t.Helper()
	c, e := Parse(fixtureBytes(t, file), 3)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func encoded(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func fixtureOptions(t *testing.T) Options {
	t.Helper()
	o := Options{Version: 4}
	m := fixture(t, "manifest.json")
	for _, v := range list(m["source_pins"]) {
		p := obj(v)
		kind := "v3-card"
		if p["file"] == "legacy-service-source.json" {
			kind = "service-descriptor"
		}
		o.Sources = append(o.Sources, Source{kind, str(p["revision"]), str(p["sha256"]), fixtureBytes(t, str(p["file"]))})
	}
	decodeReservations(t, fixture(t, "expectations.json")["reviewed_reservation_premises"], &o)
	return o
}
func decodeReservations(t *testing.T, v any, o *Options) {
	t.Helper()
	if e := json.Unmarshal(encoded(t, v), &o.Reservations); e != nil {
		t.Fatal(e)
	}
}

// These supplied complete documents satisfy the static contract; they grant no
// account/service/session admission and do not exercise a converter.
func TestCompleteCards(t *testing.T) {
	for _, name := range []string{"converted-v4.json", "pilot-v4.json", "rollback-v4.json", "remote-v4.json", "services-v4.json", "compat-id-converted-v4.json", "compat-id-pilot-v4.json", "compat-repeated-required-converted-v4.json", "compat-cross-group-converted-v4.json", "mixed-reject-nonpilot-state.json"} {
		t.Run(name, func(t *testing.T) {
			o := fixtureOptions(t)
			if name == "compat-id-pilot-v4.json" {
				decodeReservations(t, fixture(t, "compat-id-expectations.json")["reviewed_reservation_premises"], &o)
			}
			if e := Validate(fixtureBytes(t, name), o); e != nil {
				t.Fatal(e)
			}
		})
	}
}

// Counts are independent recorded v3 reader outcomes. All six accepted sources
// remain valid without v4 id/selector/collision restrictions.
func TestV3Sources(t *testing.T) {
	for _, name := range []string{"source-v3.json", "compat-id-source-v3.json", "compat-repeated-required-source-v3.json", "compat-cross-group-source-v3.json", "legacy-wrong-id-source-v3.json", "legacy-duplicate-source-v3.json"} {
		t.Run(name, func(t *testing.T) {
			raw := fixtureBytes(t, name)
			if e := Validate(raw, Options{Version: 3}); e != nil {
				t.Fatal(e)
			}
			c, _ := Parse(raw, 3)
			if len(list(c["members"])) != 4 || len(list(c["projects"])) != 3 || len(obj(c["profiles"])) != 1 {
				t.Fatal("recorded source counts differ")
			}
		})
	}
}

// This optional lane reads the explicitly supplied live v3 source without
// printing or copying its facts into test artifacts.
func TestV3SuppliedCard(t *testing.T) {
	path := os.Getenv("CARD_V3_INTEGRATION_INPUT")
	if path == "" {
		t.Skip("external source not selected")
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal("integration input unreadable")
	}
	if e := Validate(raw, Options{Version: 3}); e != nil {
		t.Fatal(e)
	}
	c, _ := Parse(raw, 3)
	if len(list(c["members"])) != 9 || len(list(c["projects"])) != 8 || len(obj(c["profiles"])) != 4 {
		t.Fatal("recorded live source counts differ")
	}
}
