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
	if entry7.InitChannels != 4 || entry7.DrawChannels != 3 || entry7.ModeReturns["3"] != 32 {
		t.Fatalf("command7 raw channel contract=%#v", entry7)
	}
	entry8, _ := table.Schedule(8)
	if entry8.Entry == "0x174B0" || entry8.ModeReturns["3"] != 34 || len(entry8.SampleMarkers) != 2 {
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
