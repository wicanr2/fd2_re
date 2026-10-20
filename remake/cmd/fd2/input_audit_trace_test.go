package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		{"closing", &nativeSystemEndTurnUIState{rewardMessage: message}, &nativeClassUIJob{frames: [][]byte{{1}, {2}}, restore: []byte{3}, frame: 1, drawn: true, timeline: []nativeClassUITimelineStep{{}}, elapsed: time.Second}, false},
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
		if row["system_end_turn_ui"] != (state.ui != nil) {
			t.Fatalf("%s: wrong system owner observation", state.name)
		}
		if job := state.job; job != nil {
			if row["class_ui_frame"] != float64(job.frame) || row["class_ui_frames"] != float64(len(job.frames)) ||
				row["class_ui_restore_length"] != float64(len(job.restore)) || row["class_ui_drawn"] != job.drawn ||
				row["class_ui_timeline_steps"] != float64(len(job.timeline)) || row["class_ui_elapsed_ns"] != float64(job.elapsed) {
				t.Fatalf("%s: wrong class UI observation: %v", state.name, row)
			}
		}
	}
}
