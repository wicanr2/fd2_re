package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeCommand0TargetKeepsTransientMessageOutOfRecoveredFrame(t *testing.T) {
	if g := (&Game{msg: "command", nativeCommand0Targeting: true}); g.shouldDrawTransientBattleMessage() {
		t.Fatalf("native command 0 target rendered transient message: %#v", g)
	}
	if g := (&Game{msg: "result"}); !g.shouldDrawTransientBattleMessage() {
		t.Fatal("unrelated battle result message was suppressed")
	}
	if g := (&Game{msg: "dialog", dialog: []battle.DialogLine{{}}}); g.shouldDrawTransientBattleMessage() {
		t.Fatal("dialog-owned message was rendered")
	}
}
