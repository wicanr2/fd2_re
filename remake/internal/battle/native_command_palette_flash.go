package battle

import (
	"encoding/json"
	"fmt"
	"os"
)

const NativeCommandPaletteFlashEntries = NativeCommandRecordCount

// NativeCommandPaletteFlashTable preserves the three command-indexed six-bit
// DAC tables read by FD2.EXE 0x1D6C8. The high-level command/status names are
// deliberately outside this raw presentation contract.
type NativeCommandPaletteFlashTable struct {
	SchemaVersion int                      `json:"schema_version"`
	Cycles        int                      `json:"cycles"`
	TicksPerPhase int                      `json:"ticks_per_phase"`
	SFXResource   int                      `json:"sfx_resource"`
	SFXIndex      int                      `json:"sfx_index"`
	Entries       [][3]byte                `json:"entries"`
	Source        NativeCommandFlashSource `json:"source"`
}

type NativeCommandFlashSource struct {
	File         string `json:"file"`
	Size         int    `json:"size"`
	MD5          string `json:"md5"`
	SHA256       string `json:"sha256"`
	IDAVersion   string `json:"ida_version"`
	AddressSpace string `json:"address_space"`
	RedTable     string `json:"red_table"`
	GreenTable   string `json:"green_table"`
	BlueTable    string `json:"blue_table"`
}

func LoadNativeCommandPaletteFlashTable(path string) (*NativeCommandPaletteFlashTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table NativeCommandPaletteFlashTable
	if err := json.Unmarshal(raw, &table); err != nil {
		return nil, err
	}
	if table.SchemaVersion != 1 || table.Cycles != 4 || table.TicksPerPhase != 1 ||
		table.SFXResource != 88 || table.SFXIndex != 0 || len(table.Entries) != NativeCommandPaletteFlashEntries {
		return nil, fmt.Errorf("native command palette flash metadata is invalid")
	}
	if table.Source.File != "FD2.EXE" || table.Source.Size != 357074 ||
		table.Source.MD5 != "b97caf2239a27a896069d03549d96e1e" ||
		table.Source.SHA256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		table.Source.AddressSpace != "IDA linear, image base 0" {
		return nil, fmt.Errorf("native command palette flash source identity is invalid")
	}
	for commandID, rgb := range table.Entries {
		for _, component := range rgb {
			if component > 0x3f {
				return nil, fmt.Errorf("native command palette flash command %d has non-DAC component", commandID)
			}
		}
	}
	return &table, nil
}

// NativeCommandPaletteFlashPhases returns the exact color/black phase order
// for one command. Each entry is one 0x17AA9(1) boundary in the original.
func (t *NativeCommandPaletteFlashTable) NativeCommandPaletteFlashPhases(commandID int) ([][3]byte, error) {
	if t == nil || commandID < 0 || commandID >= len(t.Entries) || t.Cycles != 4 || t.TicksPerPhase != 1 {
		return nil, fmt.Errorf("native command palette flash unavailable id=%d", commandID)
	}
	phases := make([][3]byte, 0, t.Cycles*2)
	for i := 0; i < t.Cycles; i++ {
		phases = append(phases, t.Entries[commandID], [3]byte{})
	}
	return phases, nil
}
