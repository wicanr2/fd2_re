package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type nativeAICommandProducerMap struct {
	Map   int `json:"map"`
	Units []struct {
		Camp               string `json:"camp"`
		NativeRecordByte34 int    `json:"native_record_byte34"`
		NativeRecordWord46 int    `json:"native_record_word46"`
		InitialCommandMask []int  `json:"initial_command_mask"`
	} `json:"units"`
}

func TestUnprovenNonPlayerDerivedStrikeFailsBeforeMutation(t *testing.T) {
	actor := &battle.Unit{Camp: battle.Enemy, HP: 80, MaxHP: 80, MP: 40, OnField: true}
	target := &battle.Unit{Camp: battle.Own, HP: 70, MaxHP: 70, OnField: true}
	g := &Game{st: &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}}}
	plan := &battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand,
		NativeCommandID: 24,
	}
	if err := g.executeNativeAIAction(plan); err == nil {
		t.Fatal("unproven non-player command24 did not fail closed")
	}
	if actor.HP != 80 || actor.MP != 40 || actor.Acted || target.HP != 70 {
		t.Fatalf("rejected command mutated actor=%#v target=%#v", actor, target)
	}
}

func TestFixedNonPlayerCommandProducersHaveIndexedOwners(t *testing.T) {
	paths, err := filepath.Glob("../../assets/maps/map*/map*_units.json")
	if err != nil || len(paths) != 33 {
		t.Fatalf("native map inventory paths=%d err=%v", len(paths), err)
	}
	producerIDs := map[int]bool{}
	rawIDs := map[int]bool{}
	id30Mode8 := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var source nativeAICommandProducerMap
		if err := json.Unmarshal(data, &source); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for index, unit := range source.Units {
			if unit.Camp != "enemy" && unit.Camp != "friend" {
				continue
			}
			if len(unit.InitialCommandMask) != 4 {
				t.Fatalf("map%d unit%d command mask=%v", source.Map, index, unit.InitialCommandMask)
			}
			mode := unit.NativeRecordByte34 & 0x0f
			for rawByte, value := range unit.InitialCommandMask {
				if value < 0 || value > 0xff {
					t.Fatalf("map%d unit%d command byte=%d", source.Map, index, value)
				}
				for bit := 0; bit < 8; bit++ {
					if value&(1<<bit) == 0 {
						continue
					}
					id := rawByte*8 + bit
					rawIDs[id] = true
					if id == 30 && source.Map == 13 && index == 0 && mode == 8 && unit.NativeRecordWord46 == 0 {
						id30Mode8++
					}
					if mode == 0 || mode == 1 || mode == 2 || mode == 3 || mode == 5 || mode == 9 || mode == 10 {
						producerIDs[id] = true
						if !nativeAICommandHasIndexedOwner(id) {
							t.Fatalf("map%d unit%d mode%d command%d lacks indexed owner", source.Map, index, mode, id)
						}
					}
				}
			}
		}
	}
	keys := func(values map[int]bool) []int {
		result := make([]int, 0, len(values))
		for value := range values {
			result = append(result, value)
		}
		sort.Ints(result)
		return result
	}
	wantRaw := []int{0, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 20, 21, 22, 26, 27, 30}
	wantProducers := []int{0, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 20, 21, 22, 26, 27}
	if got := keys(rawIDs); !reflect.DeepEqual(got, wantRaw) {
		t.Fatalf("fixed non-player raw command IDs=%v want=%v", got, wantRaw)
	}
	if got := keys(producerIDs); !reflect.DeepEqual(got, wantProducers) {
		t.Fatalf("fixed non-player scorer producers=%v want=%v", got, wantProducers)
	}
	if id30Mode8 != 1 || nativeAICommandHasIndexedOwner(30) {
		t.Fatalf("command30 mode8 count=%d admitted=%v", id30Mode8, nativeAICommandHasIndexedOwner(30))
	}
	for _, id := range []int{23, 24, 25, 28, 29, 30, 31} {
		if nativeAICommandHasIndexedOwner(id) {
			t.Fatalf("unproven non-player command %d was admitted", id)
		}
	}
}
