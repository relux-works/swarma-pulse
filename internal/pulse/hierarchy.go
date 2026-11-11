package pulse

import (
	"path/filepath"
	"time"
)

// projectHierarchy reads only the already supplied bytes. Neither an external
// parent nor a v4 source attestation selects another file or transport operation.
func projectHierarchy(raw []byte, revision, cardRoot, agent string, now time.Time, view string) ([]byte, error) {
	card, err := decodeQueryCard(raw, revision, false)
	if err != nil {
		return nil, err
	}
	if err := sourceBudget(0, len(raw), len(arr(card["members"]))); err != nil {
		return nil, err
	}
	source := &querySource{
		selection: sourceSelection{id: cardID(card), schema: str(card["schema"]), revision: revision, path: filepath.Join(cardRoot, "product-card.yaml")},
		raw:       raw,
		card:      card,
	}
	return projectGraph([]*querySource{source}, nil, agent, now, view)
}
