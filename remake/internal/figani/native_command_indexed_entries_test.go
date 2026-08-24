package figani

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNativeCommandIndexedEntryTablePreservesCorrectedLinearEntries(t *testing.T) {
	table, err := LoadNativeCommandIndexedEntryTable("../../assets/data/native_command_indexed_entries.json")
	if err != nil {
		t.Fatal(err)
	}
	for commandID, want := range nativeCommandIndexedEntries {
		entry, err := table.Schedule(commandID)
		if err != nil || entry.Entry != want {
			t.Fatalf("command %d entry=%#v err=%v want=%s", commandID, entry, err, want)
		}
	}
	entry7, _ := table.Schedule(7)
	if entry7.InitChannels != 4 || entry7.DrawChannels != 3 || entry7.ModeReturns["3"] != 32 || len(entry7.SampleMarkers) != 6 {
		t.Fatalf("command7 raw channel contract=%#v", entry7)
	}
	entry3, _ := table.Schedule(3)
	if entry3.RawSideZeroXShift != 20 || entry3.UsesRNG || len(entry3.StateRanges) != 1 ||
		entry3.StateRanges[0] != "0x53F81..0x53FE3" || len(entry3.SampleMarkers) != 6 {
		t.Fatalf("command3 direct-instruction correction=%#v", entry3)
	}
	entry8, _ := table.Schedule(8)
	if entry8.Entry == "0x174B0" || entry8.ModeReturns["3"] != 34 ||
		len(entry8.StateRanges) != 1 || entry8.StateRanges[0] != "0x540BA..0x540F9" || len(entry8.SampleMarkers) != 4 {
		t.Fatalf("command8 address-base correction=%#v", entry8)
	}
	entry2, _ := table.Schedule(2)
	if len(entry2.OffsetTables) != 0 || len(entry2.StateRanges) != 1 || len(entry2.SampleMarkers) != 3 {
		t.Fatalf("command2 corrected ownership=%#v", entry2)
	}
	entry6, _ := table.Schedule(6)
	if entry6.ModeReturns["0"] != 7 || entry6.ModeReturns["3"] != 12 || entry6.ModeReturns["6"] != 7 ||
		entry6.InitChannels != 5 || entry6.DrawChannels != 5 || len(entry6.SampleMarkers) != 3 {
		t.Fatalf("command6 direct-instruction contract=%#v", entry6)
	}
	entry5, _ := table.Schedule(5)
	if entry5.ModeReturns["0"] != 1 || entry5.ModeReturns["3"] != 12 || entry5.ModeReturns["6"] != 8 ||
		entry5.InitChannels != 6 || entry5.DrawChannels != 6 || !entry5.UsesRNG || len(entry5.SampleMarkers) != 6 {
		t.Fatalf("command5 direct-instruction contract=%#v", entry5)
	}
}

func TestLoadNativeCommandIndexedEntryTableRejectsRawUnrelocatedAddress(t *testing.T) {
	raw, err := os.ReadFile("../../assets/data/native_command_indexed_entries.json")
	if err != nil {
		t.Fatal(err)
	}
	for index := range raw {
		if index+7 <= len(raw) && string(raw[index:index+7]) == "0x274B0" {
			copy(raw[index:index+7], []byte("0x174B0"))
			break
		}
	}
	path := filepath.Join(t.TempDir(), "entries.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNativeCommandIndexedEntryTable(path); err == nil {
		t.Fatal("raw unrelocated entry address was accepted")
	}
}
