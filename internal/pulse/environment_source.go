package pulse

import (
	"time"

	"github.com/relux-works/swarma-pulse/internal/cardv4"
)

// Only the explicit environment selector admits v5. The initial parse selects
// a grammar; strict validation always reparses the exact original source bytes.
func decodeEnvironmentCard(raw []byte, revision string, now time.Time, target bool) (object, error) {
	c, issue := cardv4.Parse(raw, 3)
	if issue != nil {
		return nil, refusal("card_invalid")
	}
	if c["schema"] == "product-card/v5" {
		c, issue = cardv4.ParseGovernanceSnapshot(raw, now)
		if issue != nil {
			return nil, refusal("card_invalid")
		}
		return c, nil
	}
	if target {
		return nil, refusal("unsupported_schema")
	}
	return decodeQueryCard(raw, revision, true)
}
