package main

import "testing"

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
