package campaign

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// LoadAICommandSpellMap extracts the spell commands available from the
// editable 23-byte EXE item rows.  The dump format is
// TY,AP,HT,DP,EV,S1,S2,R1,R2,K1..K6,MM (23 bytes); command-bearing K4 is
// therefore raw byte 0x11 (the old RE note calling it +0x10 used a
// one-based/field-relative offset).  We intentionally expose only command
// bytes >= 0x10: the native AI converts those to spell_id=command-0x10, while
// lower bytes are physical/status commands and must not be treated as spells.
func LoadAICommandSpellMap(path string) (map[int]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID  int    `json:"id"`
		Raw string `json:"raw"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make(map[int]int)
	for _, row := range rows {
		if row.ID < 0 || row.ID > 0xff {
			return nil, fmt.Errorf("invalid item id %d", row.ID)
		}
		if row.Raw == "" {
			return nil, fmt.Errorf("item %d has no raw record", row.ID)
		}
		b, err := hex.DecodeString(row.Raw)
		if err != nil {
			return nil, fmt.Errorf("item %d raw: %w", row.ID, err)
		}
		if len(b) != 23 {
			return nil, fmt.Errorf("item %d raw length=%d, want 23", row.ID, len(b))
		}
		command := int(b[0x11])
		if command >= 0x10 {
			out[row.ID] = command - 0x10
		}
	}
	return out, nil
}
