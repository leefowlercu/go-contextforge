package contextforge

import (
	"encoding/json"
	"fmt"
)

// decodeListResponse decodes either:
// 1. A plain JSON array: [...]
// 2. A paginated wrapper: {"<key>":[...], "nextCursor":"..."}
func decodeListResponse[T any](raw json.RawMessage, key string) ([]*T, string, error) {
	var list []*T
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, "", nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, "", fmt.Errorf("decode list response: %w", err)
	}

	itemsRaw, ok := envelope[key]
	if !ok {
		return nil, "", fmt.Errorf("decode list response: missing %q field", key)
	}
	if err := json.Unmarshal(itemsRaw, &list); err != nil {
		return nil, "", fmt.Errorf("decode list response items: %w", err)
	}

	var nextCursor string
	if nextRaw, ok := envelope["nextCursor"]; ok {
		_ = json.Unmarshal(nextRaw, &nextCursor)
	}

	return list, nextCursor, nil
}
