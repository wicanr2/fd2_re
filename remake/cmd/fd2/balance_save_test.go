package main

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestBalanceWarningRequiresThreeSeparateClicksEveryOpening(t *testing.T) {
	g := &Game{camp: &campaign.Runner{}}
	g.openBalanceEditor()
	if !g.balanceUI.open || g.balanceUI.ack != 0 {
		t.Fatal("warning missing")
	}
	for i := 1; i <= 3; i++ {
		g.balancePoint(image.Pt(400, 264))
		if g.balanceUI.ack != i {
			t.Fatal("warning skipped a step")
		}
	}
	g.balanceUI = balanceEditor{}
	g.openBalanceEditor()
	if g.balanceUI.ack != 0 {
		t.Fatal("reopen bypassed warning")
	}
}

func TestBalanceNativeSidecarKeepsSlotsIndependentAndOriginalBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	plain := bytes.Repeat([]byte{0x5a}, fdsave.FileSize)
	stored, err := fdsave.Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, stored, 0600); err != nil {
		t.Fatal(err)
	}
	g := &Game{}
	if err = g.replaceNativeSaveWithBalance(path, stored, 0); err != nil {
		t.Fatal(err)
	}
	if fileExistsBalance(path + ".fd2-options.json") {
		t.Fatal("default created sidecar")
	}
	if raw, _ := os.ReadFile(path); !bytes.Equal(raw, stored) {
		t.Fatal("default changed native bytes")
	}
	g.balance = battle.NewBalanceOptions()
	g.balance.EnemyPercent = 200
	if err = g.replaceNativeSaveWithBalance(path, stored, 0); err != nil {
		t.Fatal(err)
	}
	one, err := loadBalanceSavedState(path, plain, 0)
	if err != nil || one.Options == nil || one.Options.EnemyPercent != 200 {
		t.Fatal("slot1 options absent", err)
	}
	two, err := loadBalanceSavedState(path, plain, 1)
	if err != nil || two.Options != nil {
		t.Fatal("slot1 leaked into slot2")
	}
	changed := append([]byte(nil), plain...)
	a, _, _ := fdsave.SlotBounds(1)
	changed[a]++
	stored2, err := fdsave.Encode(changed)
	if err != nil {
		t.Fatal(err)
	}
	g.balance.EnemyPercent = 50
	if err = g.replaceNativeSaveWithBalance(path, stored2, 1); err != nil {
		t.Fatal(err)
	}
	one, _ = loadBalanceSavedState(path, changed, 0)
	two, _ = loadBalanceSavedState(path, changed, 1)
	if one.Options.EnemyPercent != 200 || two.Options.EnemyPercent != 50 {
		t.Fatal("saving another slot invalidated options")
	}
	changed[a]++
	absent, _ := loadBalanceSavedState(path, changed, 1)
	if absent.Options != nil {
		t.Fatal("unrelated old save inherited options")
	}
}
