package main

import (
	"bytes"
	"image"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestBalanceAuditSelectionDoesNotNormalizeOrChangeState(t *testing.T) {
	u := battle.Unit{}
	u.NativeCommandMask[0] = 1
	g := &Game{partyRoster: map[int]battle.Unit{0: {}, 9: u}, rng: rand.New(rand.NewSource(1995))}
	g.balanceUI = balanceEditor{open: true, ack: 3, tab: 2, person: -1, command: 99, field: -1}
	before := g.balanceUI
	state := g.balanceAuditState()
	if state["identity"] != 9 || state["command_id"] != 0 {
		t.Fatal("audit did not describe the wrapped selection", state)
	}
	if !reflect.DeepEqual(before, g.balanceUI) || g.balance != nil || g.partyRoster[9].NativeCommandMask[0] != 1 {
		t.Fatal("read-only audit changed editor or party state")
	}
	if g.rng.Int63() != rand.New(rand.NewSource(1995)).Int63() {
		t.Fatal("read-only audit consumed gameplay RNG")
	}
}

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

func TestBalanceNativeCurrentRestoresEnemyBaselineWithoutScalingTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	plain := make([]byte, fdsave.FileSize)
	stored, err := fdsave.Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, stored, 0600); err != nil {
		t.Fatal(err)
	}
	base := battle.BalanceStats{HP: 100, MP: 20, AP: 50, DP: 40, DX: 10}
	u := &battle.Unit{Camp: battle.Enemy, HP: 80, MaxHP: 200, MP: 20, MaxMP: 40, AP: 100, DP: 80, DX: 20, EnemyBalanceBase: &base, EnemyBalancePercent: 200}
	g := &Game{st: &battle.State{Units: []*battle.Unit{u}}, balance: battle.NewBalanceOptions()}
	g.balance.EnemyPercent = 200
	if err = g.replaceNativeSaveWithBalance(path, stored, -1); err != nil {
		t.Fatal(err)
	}
	saved, err := loadBalanceSavedState(path, plain, -1)
	if err != nil {
		t.Fatal(err)
	}
	loaded := &battle.Unit{Camp: battle.Enemy, HP: 80, MaxHP: 200, MP: 20, MaxMP: 40, AP: 100, DP: 80, DX: 20}
	cold := &Game{st: &battle.State{Units: []*battle.Unit{loaded}}}
	cold.restoreBalanceSavedState(saved)
	cold.syncBalanceState()
	if loaded.MaxHP != 200 || loaded.HP != 80 || loaded.AP != 100 {
		t.Fatal("native current strength compounded")
	}
	cold.balance = nil
	cold.syncBalanceState()
	if loaded.MaxHP != 100 || loaded.HP != 40 || loaded.AP != 50 {
		t.Fatal("native current baseline was not restored")
	}
	cold.restoreBalanceSavedState(balanceSavedState{})
	if cold.balance != nil {
		t.Fatal("old save inherited prior profile")
	}
}
