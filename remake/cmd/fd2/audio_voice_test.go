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

func TestFMAndMT32TracksDecodeAndSwitchAtRuntime(t *testing.T) {
	t.Setenv("FD2_MUTE", "")
	if _, err := os.Stat(assetPath("assets/music_fm/FDMUS_001.ogg")); err != nil {
		t.Skipf("分離音樂 render 未安裝：%v", err)
	}
	for _, profile := range []string{"fm", "mt32"} {
		g := &Game{bgmSource: profile}
		for _, track := range []string{"FDMUS_004", "FDMUS_018"} {
			path, err := g.musicRenderPath(track)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatal(err)
			}
			g.playBGMCount(track, 0)
			if g.bgm == nil || g.bgmCur != track {
				t.Fatalf("%s track player=%p current=%q, want %s", profile, g.bgm, g.bgmCur, track)
			}
			g.stopBGM()
		}
		g.closeAudioPlayers()
	}
}

func TestUnknownCatalogTrackDoesNotMutateCurrentBGM(t *testing.T) {
	t.Setenv("FD2_MUTE", "")
	g := &Game{bgmSource: "fm", bgmCur: "FDMUS_004"}
	g.playBGMCount("FDMUS_999", 0)
	if g.bgm != nil || g.bgmCur != "FDMUS_004" {
		t.Fatalf("failed track changed player=%p current=%q", g.bgm, g.bgmCur)
	}
}
