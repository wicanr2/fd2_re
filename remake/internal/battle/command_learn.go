package battle

import (
	"encoding/json"
	"fmt"
	"os"
)

// CommandLearnEntry is one exact (required level, command ID) pair from the
// 12-byte native learning row selected by growth-row learn_idx.
type CommandLearnEntry struct {
	RequiredLevel int `json:"required_level"`
	CommandID     int `json:"command_id"`
}

type commandLearnFileRow struct {
	Idx     int                 `json:"idx"`
	Entries []CommandLearnEntry `json:"entries"`
}

type commandLearnSelectorRow struct {
	Idx      int `json:"idx"`
	LearnIdx int `json:"learn_idx"`
}

// LoadCommandLearn reads the editable export of 0x626b3 + learn_idx*12.
// Rows must be dense and pairs strictly increasing by level so malformed
// authored data cannot silently grant a different native command.
func LoadCommandLearn(path string) (map[int][]CommandLearnEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []commandLearnFileRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("command learn: empty table")
	}
	out := make(map[int][]CommandLearnEntry, len(rows))
	for i, row := range rows {
		if row.Idx != i || len(row.Entries) > 6 {
			return nil, fmt.Errorf("command learn: invalid row %d", i)
		}
		previous := 0
		for _, entry := range row.Entries {
			if entry.RequiredLevel <= previous || entry.CommandID < 0 || entry.CommandID >= 40 {
				return nil, fmt.Errorf("command learn: invalid entry in row %d", i)
			}
			previous = entry.RequiredLevel
		}
		out[row.Idx] = append([]CommandLearnEntry(nil), row.Entries...)
	}
	return out, nil
}

// LoadCommandLearnSelectors reads the 11-byte 0x4E4D1 growth records and
// preserves their final learn_idx byte. Native level-up does not index the
// command table with portrait: it first resolves unit+7 through this table,
// then uses byte10 as the 0x4E4A2 command-learn row. 0xFF means no row.
func LoadCommandLearnSelectors(path string) (map[int]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []commandLearnSelectorRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("command learn selectors: empty growth table")
	}
	out := make(map[int]int, len(rows))
	for i, row := range rows {
		if row.Idx != i || row.LearnIdx < 0 || row.LearnIdx > 0xff {
			return nil, fmt.Errorf("command learn selectors: invalid row %d", i)
		}
		if row.LearnIdx != 0xff {
			out[row.Idx] = row.LearnIdx
		}
	}
	return out, nil
}

func (s *State) learnNativeCommandsAtLevel(u *Unit) []int {
	if s == nil || u == nil || s.CommandLearn == nil || s.CommandLearnSelectors == nil ||
		!u.HasBattleFig {
		return nil
	}
	learnIdx, ok := s.CommandLearnSelectors[u.BattleFig]
	if !ok {
		return nil
	}
	entries, ok := s.CommandLearn[learnIdx]
	if !ok {
		return nil
	}
	var learned []int
	for _, entry := range entries {
		if entry.RequiredLevel == u.Lv && u.EnableNativeCommand(entry.CommandID) {
			learned = append(learned, entry.CommandID)
		}
	}
	return learned
}
