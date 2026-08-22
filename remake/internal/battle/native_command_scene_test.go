package battle

import "testing"

func TestLoadNativeCommandSceneTableMatches52363(t *testing.T) {
	table, err := LoadNativeCommandSceneTable("../../assets/data/native_command_scene.json")
	if err != nil {
		t.Fatal(err)
	}
	for chapter := 0; chapter < NativeCommandSceneChapterCount; chapter++ {
		got, err := table.InitialBackground(chapter)
		if err != nil {
			t.Fatal(err)
		}
		want := byte(3)
		if chapter == 24 || chapter == 25 || chapter == 26 || chapter == 28 || chapter == 29 {
			want = 0
		}
		if got != want {
			t.Fatalf("chapter %d selector=%d want=%d", chapter, got, want)
		}
	}
	if _, err := table.InitialBackground(30); err == nil {
		t.Fatal("chapter beyond fixed table accepted")
	}
}

func TestNativeCommandBackgroundGateMatches1F183(t *testing.T) {
	unit := &Unit{BattleFig: 1, HasBattleFig: true, NativeRecordRace: 4, HasNativeRecordRace: true, NativeRecordClass: 0, HasNativeRecordClass: true}
	if got, err := NativeCommandBackgroundGate(unit); err != nil || !got {
		t.Fatalf("race4 gate=%v err=%v", got, err)
	}
	unit.NativeRecordRace, unit.NativeRecordClass = 0, 19
	if got, err := NativeCommandBackgroundGate(unit); err != nil || !got {
		t.Fatalf("class19 gate=%v err=%v", got, err)
	}
	unit.BattleFig = 28
	if got, err := NativeCommandBackgroundGate(unit); err != nil || got {
		t.Fatalf("selector28 gate=%v err=%v", got, err)
	}
	unit.HasNativeRecordRace = false
	if _, err := NativeCommandBackgroundGate(unit); err == nil {
		t.Fatal("missing raw race provenance accepted")
	}
}
