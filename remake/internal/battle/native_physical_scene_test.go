package battle

import (
	"reflect"
	"testing"
)

func physicalSceneUnit(side byte, figure int, race, class byte) *Unit {
	return &Unit{HasNativeRecordByte6: true, NativeRecordByte6: side,
		HasBattleFig: true, BattleFig: figure,
		HasNativeRecordRace: true, NativeRecordRace: race,
		HasNativeRecordClass: true, NativeRecordClass: class}
}

func TestNativePhysicalSceneChapter12NormalTuple(t *testing.T) {
	actor := physicalSceneUnit(2, 4, 1, 3)
	target := physicalSceneUnit(0, 102, 8, 2)
	beforeActor, beforeTarget := *actor, *target
	got, err := SelectNativePhysicalScene(0, actor, target, [4]byte{0, 2, 5, 0}, [4]byte{0, 0, 50, 0}, 0)
	want := NativePhysicalSceneSelection{PrimaryBG: 50, SecondaryBG: 5, BaseBG: 50, TAI: 5}
	if err != nil || got != want {
		t.Fatalf("normal raw tuple = %+v / %v, want %+v", got, err, want)
	}
	if !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
		t.Fatal("scene selection modified raw units")
	}
}

func TestNativePhysicalSceneSidesAndPreludeBase(t *testing.T) {
	for _, side := range []byte{0, 1, 2, 255} {
		for _, flag := range []byte{0, 1, 2} {
			actor, target := physicalSceneUnit(side, 4, 1, 3), physicalSceneUnit(2, 102, 8, 2)
			got, err := SelectNativePhysicalScene(3, actor, target, [4]byte{0, 0, 5, 0}, [4]byte{0, 0, 50, 0}, flag)
			want := NativePhysicalSceneSelection{PrimaryBG: 5, SecondaryBG: 50, BaseBG: 5, TAI: 50, HasSeparateBackgrounds: flag != 0}
			if side != 0 {
				want.PrimaryBG, want.SecondaryBG, want.TAI = 50, 5, 5
				if flag == 0 {
					want.BaseBG = 50
				}
			}
			if err != nil || got != want {
				t.Fatalf("raw side=%d flag=%d: %+v / %v, want %+v", side, flag, got, err, want)
			}
		}
	}
}

func TestNativePhysicalSceneGateAndZeroInitialTAI(t *testing.T) {
	cases := []struct {
		figure      int
		race, class byte
		gate        bool
	}{
		{27, 3, 19, true}, {29, 4, 18, true}, {29, 5, 18, true},
		{29, 3, 18, false}, {29, 6, 18, false}, {28, 4, 19, false},
	}
	for _, tc := range cases {
		for _, initial := range []byte{0, 3} {
			actor := physicalSceneUnit(2, tc.figure, tc.race, tc.class)
			target := physicalSceneUnit(0, 102, 8, 2)
			got, err := SelectNativePhysicalScene(initial, actor, target, [4]byte{0, 0, 5, 0}, [4]byte{0, 0, 50, 0}, 0)
			wantBG, wantTAI := byte(5), byte(5)
			if tc.gate {
				wantTAI = initial
				if initial != 0 {
					wantBG = initial
				}
			}
			if err != nil || got.PrimaryBG != 50 || got.SecondaryBG != wantBG || got.TAI != wantTAI {
				t.Fatalf("gate case=%+v initial=%d: %+v / %v, want BG=%d TAI=%d", tc, initial, got, err, wantBG, wantTAI)
			}
			// 同一 gate 放在 primary，仍須排除 figure28，initial0時取地形。
			actor.NativeRecordByte6 = 0
			got, err = SelectNativePhysicalScene(initial, actor, target, [4]byte{0, 0, 5, 0}, [4]byte{0, 0, 50, 0}, 0)
			if err != nil || got.PrimaryBG != wantBG || got.SecondaryBG != 50 || got.TAI != 50 {
				t.Fatalf("primary gate case=%+v initial=%d: %+v / %v", tc, initial, got, err)
			}
		}
	}
}

func TestNativePhysicalSceneMissingRawProvenanceRejects(t *testing.T) {
	mutations := []func(*Unit){
		func(u *Unit) { u.HasNativeRecordByte6 = false },
		func(u *Unit) { u.HasBattleFig = false },
		func(u *Unit) { u.HasNativeRecordRace = false },
		func(u *Unit) { u.HasNativeRecordClass = false },
		func(u *Unit) { u.BattleFig = -1 },
		func(u *Unit) { u.BattleFig = 256 },
	}
	for _, mutate := range mutations {
		for _, changeActor := range []bool{false, true} {
			actor, target := physicalSceneUnit(2, 4, 1, 3), physicalSceneUnit(0, 102, 8, 2)
			if changeActor {
				mutate(actor)
			} else {
				mutate(target)
			}
			if _, err := SelectNativePhysicalScene(0, actor, target, [4]byte{}, [4]byte{}, 0); err == nil {
				t.Fatal("missing raw input accepted")
			}
		}
	}
	if _, err := SelectNativePhysicalScene(4, physicalSceneUnit(2, 4, 1, 3), physicalSceneUnit(0, 102, 8, 2), [4]byte{}, [4]byte{}, 0); err == nil {
		t.Fatal("initial outside verified chapter table accepted")
	}
}
