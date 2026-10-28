package main

import (
	"testing"
	"time"
)

func TestPlayerGameClockChangesSpeedWithoutJump(t *testing.T) {
	start := time.Unix(1000, 0)
	g := &Game{}
	g.setPlayerSpeedAt(1, start)
	if got := g.playerClock.sample(start.Add(time.Second)); !got.Equal(start.Add(time.Second)) {
		t.Fatal(got)
	}
	g.setPlayerSpeedAt(4, start.Add(2*time.Second))
	if !g.playerClock.game.Equal(start.Add(2 * time.Second)) {
		t.Fatal("speed change jumped time")
	}
	if got := g.playerClock.sample(start.Add(3 * time.Second)); !got.Equal(start.Add(6 * time.Second)) {
		t.Fatal(got)
	}
	g.setPlayerSpeedAt(1, start.Add(3*time.Second))
	if got := g.playerClock.sample(start.Add(4 * time.Second)); !got.Equal(start.Add(7 * time.Second)) {
		t.Fatal(got)
	}
	if got := g.playerClock.sample(start); !got.Equal(start.Add(7 * time.Second)) {
		t.Fatal("wall regression rewound gameplay")
	}
	for _, bad := range []int{-1, 0, 5, 1000000} {
		if normalizePlayerSpeed(bad) != 1 {
			t.Fatal("invalid speed accepted")
		}
	}
}

func TestPlayerSpeedIsSeparateFromCampaign(t *testing.T) {
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = oldCache })
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSettings(settings{BGMSource: "fm", LocaleID: "en", InterfaceMode: playerInterfaceHD, ThemeID: "faithful", FontScale: 1.25})
	g := &Game{}
	g.cyclePlayerSpeed()
	s := loadSettings()
	if s.GameSpeed != 2 || s.InterfaceMode != playerInterfaceHD || s.ThemeID != "faithful" || s.FontScale != 1.25 {
		t.Fatal("speed erased another preference", s)
	}
	if g.st != nil || g.camp != nil || g.playerSpeed != 2 {
		t.Fatal("speed created campaign state")
	}
}
