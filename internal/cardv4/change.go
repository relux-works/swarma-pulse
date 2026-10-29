package cardv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Snapshot carries actual bytes and an externally admitted expected digest.
// A matching caller-supplied hash is integrity evidence, not custody admission.
type Snapshot struct {
	Bytes  []byte
	SHA256 string
}

// CheckChange validates three v4 snapshots, checks the existing pilot-only
// preservation contract and requires byte-exact prior-card rollback. It performs
// no effects and proves no registration, custody, cutover or recovery operation.
func CheckChange(prior, candidate, rollback Snapshot, pilot string, options Options) *Issue {
	options.Version = 4
	var cards [3]Object
	for i, snapshot := range []Snapshot{prior, candidate, rollback} {
		at := index("$", i)
		if len(snapshot.Bytes) > MaxBytes {
			return shape("raw_byte_limit", at)
		}
		if !digest.MatchString(snapshot.SHA256) {
			return shape("snapshot_pin_invalid", at)
		}
		sum := sha256.Sum256(snapshot.Bytes)
		if hex.EncodeToString(sum[:]) != snapshot.SHA256 {
			return fail("snapshot_digest_mismatch", at)
		}
		if issue := Validate(snapshot.Bytes, options); issue != nil {
			return &Issue{issue.Rule, at + strings.TrimPrefix(issue.Location, "$"), issue.Exit}
		}
		cards[i], _ = Parse(snapshot.Bytes, 4)
	}
	if issue := VerifyPilotPreservation(cards[0], cards[1], pilot); issue != nil {
		return issue
	}
	if !bytes.Equal(prior.Bytes, rollback.Bytes) {
		return fail("rollback_bytes_mismatch", "$[2]")
	}
	return nil
}
