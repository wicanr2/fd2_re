package main

import (
	"os"
	"path/filepath"
	"testing"
)

type fakeSFXVoice struct {
	playing bool
	closed  bool
}

func (v *fakeSFXVoice) IsPlaying() bool { return v.playing }
func (v *fakeSFXVoice) Close() error {
	v.closed = true
	return nil
}

func TestStepSFXVoicesRetainsPlayingAndClosesFinished(t *testing.T) {
	playing := &fakeSFXVoice{playing: true}
	finished := &fakeSFXVoice{}
	g := &Game{sfxVoices: []sfxVoice{playing, nil, finished}}

	g.stepSFXVoices()

	if len(g.sfxVoices) != 1 || g.sfxVoices[0] != playing {
		t.Fatalf("active voices=%v, want only playing voice", g.sfxVoices)
	}
	if playing.closed || !finished.closed {
		t.Fatalf("close state playing=%v finished=%v", playing.closed, finished.closed)
	}
}

func TestStepSFXVoicesEventuallyReleasesAllVoices(t *testing.T) {
	voice := &fakeSFXVoice{playing: true}
	g := &Game{sfxVoices: []sfxVoice{voice}}
	g.stepSFXVoices()
	voice.playing = false
	g.stepSFXVoices()

	if len(g.sfxVoices) != 0 || !voice.closed {
		t.Fatalf("voices=%d closed=%v, want released", len(g.sfxVoices), voice.closed)
	}
}

func TestCloseAudioPlayersClosesRemainingSFXVoices(t *testing.T) {
	first := &fakeSFXVoice{playing: true}
	second := &fakeSFXVoice{playing: true}
	g := &Game{sfxVoices: []sfxVoice{first, second}}

	g.closeAudioPlayers()

	if !first.closed || !second.closed || g.sfxVoices != nil {
		t.Fatalf("closed=%v/%v voices=%v", first.closed, second.closed, g.sfxVoices)
	}
}

func TestPlayRawRetainsRealAudioPlayer(t *testing.T) {
	t.Setenv("FD2_MUTE", "")
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "sfx", "FDOTHER_031", "resource.json")); err != nil {
		t.Skipf("separated UI sound pack is absent: %v", err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	bank, err := loadSFX()
	if err != nil {
		t.Fatal(err)
	}
	pcm := bank[4]
	if len(pcm) == 0 || audioCtx == nil {
		t.Fatal("separated SFX fixture did not decode into an audio context")
	}
	g := &Game{}
	g.playRaw(pcm)
	defer g.closeAudioPlayers()

	if len(g.sfxVoices) != 1 || g.sfxVoices[0] == nil {
		t.Fatalf("retained voices=%d, want one real player", len(g.sfxVoices))
	}
}

func TestEndingMT32TracksDecodeAndSwitchAtRuntime(t *testing.T) {
	t.Setenv("FD2_MUTE", "")
	for _, track := range []string{"FDMUS_004", "FDMUS_018"} {
		if _, err := os.Stat(musicPath("mt32", track)); err != nil {
			t.Skipf("玩家自備 MT-32 OGG 不存在：%v", err)
		}
	}
	g := &Game{bgmSource: "mt32"}
	defer g.closeAudioPlayers()

	g.playBGMCount("FDMUS_004", 0)
	if g.bgm == nil || g.bgmCur != "FDMUS_004" {
		t.Fatalf("party-cycle track player=%p current=%q", g.bgm, g.bgmCur)
	}
	g.stopBGM()
	if g.bgm != nil || g.bgmCur != "" {
		t.Fatalf("ending stop left player=%p current=%q", g.bgm, g.bgmCur)
	}
	g.playBGMCount("FDMUS_018", 0)
	if g.bgm == nil || g.bgmCur != "FDMUS_018" {
		t.Fatalf("tail track player=%p current=%q", g.bgm, g.bgmCur)
	}
}
