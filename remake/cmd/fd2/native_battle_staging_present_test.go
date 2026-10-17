package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestChapter30BattleStagingMissingSourceDoesNotPublish(t *testing.T) {
	actor := &battle.Unit{X: 1, Y: 2}
	g := &Game{st: &battle.State{Units: []*battle.Unit{actor}}, curX: 7, curY: 8}
	if err := g.startNativeBattleStagingPresent(&battle.NativeStagingPlacement{Slot: 0, X: 3, Y: 4}, nil); err == nil {
		t.Fatal("缺原生視圖及圖像來源仍啟動")
	}
	if actor.X != 1 || actor.Y != 2 || g.curX != 7 || g.curY != 8 || g.focusJob != nil || g.nativeUnitPresent != nil {
		t.Fatal("失敗時留下座標或演出半成品")
	}
}
