package battle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const NativeCommandSceneChapterCount = 30

// NativeCommandSceneTable preserves the command-presentation background
// initializer at 0x52363. It remains editable data but is accepted only when
// bound to the fixed reference executable and complete chapter range.
type NativeCommandSceneTable struct {
	SchemaVersion            int                      `json:"schema_version"`
	ChapterInitialBackground []byte                   `json:"chapter_initial_background"`
	Source                   NativeCommandSceneSource `json:"source"`
}

type NativeCommandSceneSource struct {
	File                 string `json:"file"`
	Size                 int    `json:"size"`
	MD5                  string `json:"md5"`
	SHA256               string `json:"sha256"`
	IDAVersion           string `json:"ida_version"`
	AddressSpace         string `json:"address_space"`
	InitialSelectorTable string `json:"initial_selector_table"`
	RawGate              string `json:"raw_gate"`
}

func LoadNativeCommandSceneTable(path string) (*NativeCommandSceneTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table NativeCommandSceneTable
	if err := json.Unmarshal(raw, &table); err != nil {
		return nil, err
	}
	if table.SchemaVersion != 1 || len(table.ChapterInitialBackground) != NativeCommandSceneChapterCount ||
		table.Source.File != "FD2.EXE" || table.Source.Size != 357074 ||
		table.Source.MD5 != "b97caf2239a27a896069d03549d96e1e" ||
		table.Source.SHA256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		table.Source.AddressSpace != "IDA linear, image base 0" ||
		table.Source.InitialSelectorTable != "0x52363..0x52380" ||
		table.Source.RawGate != "sub_1F183 0x1F183..0x1F1BE" {
		return nil, errors.New("native command scene metadata is invalid")
	}
	for chapter, selector := range table.ChapterInitialBackground {
		if selector > 3 {
			return nil, fmt.Errorf("native command scene chapter %d selector is invalid", chapter)
		}
	}
	return &table, nil
}

func (t *NativeCommandSceneTable) InitialBackground(chapter int) (byte, error) {
	if t == nil || chapter < 0 || chapter >= len(t.ChapterInitialBackground) {
		return 0, fmt.Errorf("native command scene chapter unavailable: %d", chapter)
	}
	return t.ChapterInitialBackground[chapter], nil
}

// NativeCommandBackgroundGate reproduces sub_1F183 from raw runtime fields.
// It deliberately requires independent +7/+0x1f/+0x20 provenance and does
// not substitute normalized class labels or map selectors.
func NativeCommandBackgroundGate(unit *Unit) (bool, error) {
	if unit == nil || !unit.HasBattleFig || !unit.HasNativeRecordRace || !unit.HasNativeRecordClass {
		return false, errors.New("native command background gate raw record unavailable")
	}
	if unit.BattleFig == 28 {
		return false, nil
	}
	return unit.NativeRecordClass == 19 || unit.NativeRecordRace == 4 || unit.NativeRecordRace == 5, nil
}
