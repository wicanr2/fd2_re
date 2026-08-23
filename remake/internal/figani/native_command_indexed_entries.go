package figani

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type NativeCommandIndexedEntrySource struct {
	File         string `json:"file"`
	Size         int    `json:"size"`
	MD5          string `json:"md5"`
	SHA256       string `json:"sha256"`
	IDAVersion   string `json:"ida_version"`
	AddressSpace string `json:"address_space"`
	EntryTable   string `json:"entry_table"`
	Consumer     string `json:"consumer"`
}

type NativeCommandIndexedSampleMarker struct {
	Mode    int    `json:"mode"`
	Channel int    `json:"channel"`
	Counter int    `json:"counter"`
	Callee  string `json:"callee"`
}

// NativeCommandIndexedEntrySchedule 保存原版 funcs_2AC25 的逐 entry raw
// mode 契約；欄位只描述排程，不替畫面、音效或狀態命名。
type NativeCommandIndexedEntrySchedule struct {
	CommandID         int                                `json:"command_id"`
	Entry             string                             `json:"entry"`
	ModeReturns       map[string]int                     `json:"mode_returns"`
	InitChannels      int                                `json:"init_channels"`
	DrawChannels      int                                `json:"draw_channels"`
	DrawModes         []int                              `json:"draw_modes"`
	RawSideZeroXShift int                                `json:"raw_side_zero_x_shift"`
	OffsetTables      []string                           `json:"offset_tables"`
	StateRanges       []string                           `json:"state_ranges"`
	UsesRNG           bool                               `json:"uses_rng"`
	SampleMarkers     []NativeCommandIndexedSampleMarker `json:"sample_markers"`
}

type NativeCommandIndexedEntryTable struct {
	SchemaVersion int                                 `json:"schema_version"`
	Source        NativeCommandIndexedEntrySource     `json:"source"`
	Entries       []NativeCommandIndexedEntrySchedule `json:"entries"`
}

var nativeCommandIndexedEntries = map[int]string{
	1: "0x262EF", 2: "0x26528", 3: "0x26795", 4: "0x269D3",
	5: "0x26BFD", 6: "0x26E39", 7: "0x272B8", 8: "0x274B0",
}

func LoadNativeCommandIndexedEntryTable(path string) (*NativeCommandIndexedEntryTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table NativeCommandIndexedEntryTable
	if err := json.Unmarshal(raw, &table); err != nil {
		return nil, err
	}
	if table.SchemaVersion != 1 || table.Source.File != "FD2.EXE" || table.Source.Size != 357074 ||
		table.Source.MD5 != "b97caf2239a27a896069d03549d96e1e" ||
		table.Source.SHA256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		table.Source.AddressSpace != "IDA linear, image base 0" || table.Source.EntryTable != "0x523B9" ||
		table.Source.Consumer != "0x2A6BD" || len(table.Entries) != len(nativeCommandIndexedEntries) {
		return nil, errors.New("native command indexed entry metadata is invalid")
	}
	seen := make(map[int]bool, len(table.Entries))
	for _, entry := range table.Entries {
		want, ok := nativeCommandIndexedEntries[entry.CommandID]
		if !ok || seen[entry.CommandID] || entry.Entry != want || entry.InitChannels <= 0 ||
			entry.DrawChannels <= 0 || entry.DrawChannels > entry.InitChannels || len(entry.DrawModes) == 0 ||
			(entry.CommandID != 2 && len(entry.OffsetTables) == 0) || (entry.CommandID == 2 && len(entry.OffsetTables) != 0) {
			return nil, fmt.Errorf("native command indexed entry %d is invalid", entry.CommandID)
		}
		for _, mode := range [...]string{"0", "3", "6"} {
			if _, ok := entry.ModeReturns[mode]; !ok {
				return nil, fmt.Errorf("native command indexed entry %d mode %s is unavailable", entry.CommandID, mode)
			}
		}
		for _, mode := range entry.DrawModes {
			if mode < 1 || mode > 8 || mode == 3 || mode == 6 {
				return nil, fmt.Errorf("native command indexed entry %d draw mode %d is invalid", entry.CommandID, mode)
			}
		}
		for _, marker := range entry.SampleMarkers {
			if marker.Mode < 0 || marker.Mode > 8 || marker.Channel >= entry.DrawChannels ||
				(marker.Callee != "0x25A96" && marker.Callee != "0x25B45") {
				return nil, fmt.Errorf("native command indexed entry %d sample marker is invalid", entry.CommandID)
			}
		}
		seen[entry.CommandID] = true
	}
	return &table, nil
}

func (t *NativeCommandIndexedEntryTable) Schedule(commandID int) (NativeCommandIndexedEntrySchedule, error) {
	if t == nil {
		return NativeCommandIndexedEntrySchedule{}, errors.New("native command indexed entry table unavailable")
	}
	for _, entry := range t.Entries {
		if entry.CommandID == commandID {
			return entry, nil
		}
	}
	return NativeCommandIndexedEntrySchedule{}, fmt.Errorf("native command indexed entry %d unavailable", commandID)
}
