package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

type balanceSavedState struct {
	Options    *battle.BalanceOptions      `json:"options,omitempty"`
	EnemyBases map[int]battle.BalanceStats `json:"enemy_bases,omitempty"`
}
type balanceSidecar struct {
	Version int                          `json:"version"`
	Entries map[string]balanceSavedState `json:"entries"`
}

func balanceSaveKey(plain []byte, slot int) (string, error) {
	if len(plain) != fdsave.FileSize {
		return "", fmt.Errorf("能力設定存檔大小不符")
	}
	if slot == -1 {
		return fmt.Sprintf("current:%x", sha256.Sum256(plain[:fdsave.CurrentRuntimeHeaderOffset+fdsave.CurrentRuntimeHeaderSize])), nil
	}
	a, b, err := fdsave.SlotBounds(slot)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("slot%d:%x", slot, sha256.Sum256(plain[a:b])), nil
}

func readBalanceSidecar(path string) (balanceSidecar, error) {
	d := balanceSidecar{Version: 1, Entries: map[string]balanceSavedState{}}
	f, err := os.Open(path + ".fd2-options.json")
	if os.IsNotExist(err) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return d, err
	}
	if info.Size() > 1024*1024 {
		return d, fmt.Errorf("能力設定附檔過大")
	}
	err = json.NewDecoder(f).Decode(&d)
	if err != nil {
		return d, err
	}
	if d.Version != 1 || len(d.Entries) > 10 {
		return d, fmt.Errorf("能力設定附檔版本或筆數無效")
	}
	for _, v := range d.Entries {
		if len(v.EnemyBases) > 0 && v.Options == nil {
			return d, fmt.Errorf("敵軍基準缺少倍率")
		}
		if err = v.Options.Validate(); err != nil {
			return d, err
		}
		for i, b := range v.EnemyBases {
			if i < 0 || i >= fdsave.RosterUnits*3 || b.HP < 1 || b.HP > 32767 || b.MP < 0 || b.MP > 32767 || b.AP < 0 || b.AP > 32767 || b.DP < 0 || b.DP > 32767 || b.DX < 0 || b.DX > 32767 {
				return d, fmt.Errorf("敵軍基準無效")
			}
		}
	}
	if d.Entries == nil {
		d.Entries = map[string]balanceSavedState{}
	}
	return d, nil
}

func loadBalanceSavedState(path string, plain []byte, slot int) (balanceSavedState, error) {
	d, err := readBalanceSidecar(path)
	if err != nil {
		return balanceSavedState{}, err
	}
	key, err := balanceSaveKey(plain, slot)
	if err != nil {
		return balanceSavedState{}, err
	}
	return d.Entries[key], nil
}

// Preserve both old and new fingerprints before replacing the native SAV.
// A crash at either rename still pairs a save only with its own options.
func (g *Game) replaceNativeSaveWithBalance(path string, stored []byte, slot int) error {
	d, err := readBalanceSidecar(path)
	if err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	oldPlain, err := fdsave.Decode(old)
	if err != nil {
		return err
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, p := range [][]byte{oldPlain, plain} {
		for s := -1; s < 4; s++ {
			k, e := balanceSaveKey(p, s)
			if e != nil {
				return e
			}
			keep[k] = true
		}
	}
	for k := range d.Entries {
		if !keep[k] {
			delete(d.Entries, k)
		}
	}
	key, err := balanceSaveKey(plain, slot)
	if err != nil {
		return err
	}
	state := balanceSavedState{Options: g.balance}
	if slot == -1 && g.st != nil && g.balance != nil && g.balance.EnemyPercent != 100 {
		state.EnemyBases = map[int]battle.BalanceStats{}
		for i, u := range g.st.Units {
			if u != nil && u.Camp == battle.Enemy && u.EnemyBalanceBase != nil {
				state.EnemyBases[i] = *u.EnemyBalanceBase
			}
		}
	}
	if g.balance.IsOriginal() {
		delete(d.Entries, key)
	} else {
		d.Entries[key] = state
	}
	if len(d.Entries) > 0 || fileExistsBalance(path+".fd2-options.json") {
		raw, e := json.MarshalIndent(d, "", " ")
		if e != nil {
			return e
		}
		if e = writeSaveFile(path+".fd2-options.json", raw); e != nil {
			return e
		}
	}
	return replaceNativeCurrentSaveAtomic(path, stored)
}

func fileExistsBalance(path string) bool { _, err := os.Stat(path); return err == nil }

func (g *Game) restoreBalanceSavedState(state balanceSavedState) {
	g.balance = state.Options
	g.balanceUI = balanceEditor{}
	if g.st == nil {
		return
	}
	g.st.Balance = g.balance
	for i, b := range state.EnemyBases {
		if i >= 0 && i < len(g.st.Units) {
			u := g.st.Units[i]
			if u != nil && u.Camp == battle.Enemy {
				copy := b
				u.EnemyBalanceBase = &copy
				u.EnemyBalancePercent = g.balance.EnemyPercent
			}
		}
	}
}
