package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeHealStageMissingSourceIsAtomic(t *testing.T) {
	g, a, b, targets := healStageScenario(t, 13, false, false)
	g.nativeCh23State = nil
	before := itemHDState(g)
	calls, done := 0, 0
	err := g.startNativeCommandHealPresentation(13, targets, func() ([]battle.NativeCommandHealResult, error) { calls++; return nil, nil }, func([]battle.NativeCommandHealResult) { done++ })
	if err == nil || g.nativeHealPresentation != nil || calls != 0 || done != 0 || !reflect.DeepEqual(before, itemHDState(g)) {
		t.Fatal("missing stage committed or activated a heal", err, a.MP, b.HP)
	}
}
