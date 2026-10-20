package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInputAuditTracksDeathRewardMessageOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	t.Setenv("FD2_INPUT_AUDIT_LOG", path)
	t.Setenv("FD2_INPUT_AUDIT_INTERVAL", "1")
	message := &nativeDeathRewardMessageState{}
	states := []struct {
		name string
		ui   *nativeSystemEndTurnUIState
		job  *nativeClassUIJob
		wait bool
	}{
		{"idle", nil, nil, false},
		{"other-modal", &nativeSystemEndTurnUIState{}, nil, false},
		{"opening", &nativeSystemEndTurnUIState{rewardMessage: message}, &nativeClassUIJob{}, false},
		{"await-ack", &nativeSystemEndTurnUIState{rewardMessage: message}, nil, true},
		{"closing", &nativeSystemEndTurnUIState{rewardMessage: message}, &nativeClassUIJob{}, false},
		{"released", nil, nil, false},
	}
	g := &Game{gold: 200}
	for i, state := range states {
		g.frame = i
		g.nativeSystemEndTurnUI, g.nativeClassUIJob = state.ui, state.job
		message.awaitAck = state.wait
		g.recordInputAuditState()
		if g.nativeSystemEndTurnUI != state.ui || g.nativeClassUIJob != state.job || message.awaitAck != state.wait || g.gold != 200 {
			t.Fatalf("%s: observation changed the reward owner", state.name)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(states) {
		t.Fatalf("audit rows = %d, want %d", len(lines), len(states))
	}
	for i, line := range lines {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		state := states[i]
		active := state.ui != nil && state.ui.rewardMessage != nil
		if row["death_reward_message"] != active || row["death_reward_message_wait"] != state.wait || row["class_ui_job"] != (state.job != nil) {
			t.Fatalf("%s: message=%v wait=%v job=%v, want %v/%v/%v", state.name,
				row["death_reward_message"], row["death_reward_message_wait"], row["class_ui_job"], active, state.wait, state.job != nil)
		}
	}
}
