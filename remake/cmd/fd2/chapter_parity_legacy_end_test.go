package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type parityLegacyENDSource struct {
	EndSeq       int    `json:"end_seq"`
	BeforeSeq    int    `json:"before_seq"`
	CursorSHA256 string `json:"cursor_checkpoint_sha256"`
	SystemSHA256 string `json:"system_checkpoint_sha256"`
}

// #179 binds a missing legacy field to existing raw oracle owners. It never
// modifies the original actions or synthesizes a cursor from remake output.
func recoverParityLegacyENDSources(run string, actions []parityAction, keys map[int]string) ([]parityLegacyENDSource, error) {
	read := func(seq int) ([]byte, int, []int, []string, error) {
		raw, err := os.ReadFile(filepath.Join(run, fmt.Sprintf("checkpoint-%04d.json", seq)))
		var cp struct {
			EXE   string   `json:"exe_sha256"`
			Chain []string `json:"input_chain"`
			View  *struct {
				Round *int `json:"round"`
				X     *int `json:"cursor_x"`
				Y     *int `json:"cursor_y"`
			} `json:"view"`
		}
		if err != nil {
			return nil, 0, nil, nil, err
		}
		if err = json.Unmarshal(raw, &cp); err != nil {
			return nil, 0, nil, nil, err
		}
		if cp.View == nil || cp.View.Round == nil || cp.View.X == nil || cp.View.Y == nil {
			return nil, 0, nil, nil, fmt.Errorf("legacy END view missing")
		}
		if cp.EXE != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" {
			return nil, 0, nil, nil, fmt.Errorf("legacy END source EXE unknown")
		}
		return raw, *cp.View.Round, []int{*cp.View.X, *cp.View.Y}, cp.Chain, nil
	}
	var sources []parityLegacyENDSource
	for i := 0; i+1 < len(actions); i++ {
		clear, end := actions[i], &actions[i+1]
		if clear.Kind != "force_enemy_clear" || end.Kind != "end_turn" || end.BeforeSeq != nil {
			continue
		}
		if clear.Round != end.Round || clear.Seq >= end.Seq || len(end.At) != 2 {
			return nil, fmt.Errorf("legacy clear/END boundaries disagree")
		}
		_, round, _, chain, err := read(clear.Seq)
		if err != nil || round != clear.Round || !slices.Contains(chain, "0x117F8") {
			return nil, fmt.Errorf("legacy clear lacks normal cursor owner: %v", err)
		}
		var matches []parityLegacyENDSource
		for seq := clear.Seq + 1; seq < end.Seq; seq++ {
			if keys[seq] != "enter" && keys[seq] != "space" {
				continue
			}
			raw, currentRound, cursor, currentChain, err := read(seq)
			if err != nil {
				return nil, err
			}
			if currentRound != end.Round || !slices.Equal(cursor, end.At) ||
				!slices.Contains(currentChain, "0x118C6") || !slices.Contains(currentChain, "0x16FAE") {
				continue
			}
			previous, previousRound, _, previousChain, err := read(seq - 1)
			if err != nil {
				return nil, err
			}
			if previousRound != end.Round || !slices.Contains(previousChain, "0x117F8") {
				continue
			}
			matches = append(matches, parityLegacyENDSource{EndSeq: end.Seq, BeforeSeq: seq,
				CursorSHA256: fmt.Sprintf("%x", sha256.Sum256(previous)), SystemSHA256: fmt.Sprintf("%x", sha256.Sum256(raw))})
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("legacy END before_seq has %d proven sources", len(matches))
		}
		seq := matches[0].BeforeSeq
		end.BeforeSeq = &seq
		sources = append(sources, matches[0])
	}
	return sources, nil
}

func TestParityLegacyENDRequiresUniqueRawOwners(t *testing.T) {
	for _, bad := range []string{"", "exe", "round", "cursor", "owner", "missing", "ambiguous"} {
		t.Run(bad, func(t *testing.T) {
			run := t.TempDir()
			write := func(seq int, chain []string, exe string, round, x int) {
				cp := map[string]any{"exe_sha256": exe, "input_chain": chain, "view": map[string]int{"round": round, "cursor_x": x, "cursor_y": 4}}
				raw, _ := json.Marshal(cp)
				if err := os.WriteFile(filepath.Join(run, fmt.Sprintf("checkpoint-%04d.json", seq)), raw, 0644); err != nil {
					t.Fatal(err)
				}
			}
			exe := "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
			write(10, []string{"0x117F8"}, exe, 5, 3)
			chain := []string{"0x118C6", "0x16FAE"}
			round, x := 5, 3
			if bad == "exe" {
				exe = "unknown"
			}
			if bad == "round" {
				round = 6
			}
			if bad == "cursor" {
				x = 2
			}
			if bad == "owner" {
				chain = []string{"0x19B89"}
			}
			write(11, chain, exe, round, x)
			if bad == "missing" {
				os.Remove(filepath.Join(run, "checkpoint-0011.json"))
			}
			keys := map[int]string{11: "enter"}
			if bad == "ambiguous" {
				write(12, []string{"0x117F8"}, exe, 5, 3)
				write(13, chain, exe, 5, 3)
				keys[13] = "enter"
			}
			actions := []parityAction{{Kind: "force_enemy_clear", Seq: 10, Round: 5}, {Kind: "end_turn", Seq: 15, Round: 5, At: []int{3, 4}}}
			sources, err := recoverParityLegacyENDSources(run, actions, keys)
			if bad != "" {
				if err == nil || actions[1].BeforeSeq != nil {
					t.Fatal("unknown or ambiguous raw source was accepted")
				}
				return
			}
			if err != nil || len(sources) != 1 || actions[1].BeforeSeq == nil || *actions[1].BeforeSeq != 11 || len(sources[0].CursorSHA256) != 64 {
				t.Fatal(sources, err)
			}
			// An explicit existing source remains authoritative on a second pass.
			if repeated, err := recoverParityLegacyENDSources(run, actions, nil); err != nil || len(repeated) != 0 {
				t.Fatal("existing before_seq was overwritten")
			}
		})
	}
}
