package main

import "testing"

func TestNativeBattleDialogueAdvanceInput(t *testing.T) {
	tests := []struct {
		name         string
		enterOrSpace bool
		escape       bool
		want         bool
	}{
		{name: "none"},
		{name: "enter_or_space", enterOrSpace: true, want: true},
		{name: "escape", escape: true, want: true},
		{name: "both", enterOrSpace: true, escape: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nativeBattleDialogueAdvanceInput(tt.enterOrSpace, tt.escape); got != tt.want {
				t.Fatalf("nativeBattleDialogueAdvanceInput(%v, %v) = %v, want %v",
					tt.enterOrSpace, tt.escape, got, tt.want)
			}
		})
	}
}
