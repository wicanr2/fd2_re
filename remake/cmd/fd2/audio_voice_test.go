package main

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type fakeSFXVoice struct {
	playing bool
	closed  bool
	onClose func()
}

func (v *fakeSFXVoice) IsPlaying() bool { return v.playing }
func (v *fakeSFXVoice) Close() error {
	v.closed = true
	if v.onClose != nil {
		v.onClose()
	}
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

func TestNativePhysicalBodyCueKeepsGeneralVoicesIndependent(t *testing.T) {
	t.Setenv("FD2_MUTE", "")
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	sounds := scene.bodyResources.mainSounds
	var cue int
	for id, pcm := range sounds {
		if len(pcm) != 0 {
			cue = id
			break
		}
	}
	if len(sounds[cue]) == 0 || audioCtx == nil {
		t.Fatal("physical sound fixture did not decode PCM")
	}
	ui := &fakeSFXVoice{playing: true}
	g.sfxVoices = []sfxVoice{ui}
	scene.body = &nativePhysicalBodyPlayback{jobs: []nativePhysicalBodyJob{{cue: cue, sounds: sounds, pixels: []byte{1}}}}
	defer g.closeAudioPlayers()
	if err := g.stepNativePhysicalBodyMillis(scene, 1); err != nil {
		t.Fatal(err)
	}
	if len(g.sfxVoices) != 1 || g.sfxVoices[0] != ui || ui.closed {
		t.Fatalf("physical cue joined or interrupted general voices: count=%d UI closed=%v", len(g.sfxVoices), ui.closed)
	}
	if g.nativePhysicalVoice == nil {
		t.Fatal("physical cue lost its player owner")
	}
	// 第二個正式cue必須先關閉舊通道，不能關閉獨立UI／title。
	g.stopNativePhysicalSound()
	previous := &fakeSFXVoice{playing: true}
	title := &fakeSFXVoice{playing: true}
	g.nativePhysicalVoice, g.titleANI1Voice = previous, title
	scene.body.cued = false
	if err := g.stepNativePhysicalBodyMillis(scene, 1); err != nil {
		t.Fatal(err)
	}
	if !previous.closed || g.nativePhysicalVoice == nil || g.nativePhysicalVoice == previous || title.closed || ui.closed {
		t.Fatal("replacement did not close only the previous physical player")
	}
}

func TestNativePhysicalReturnComposesMapBeforeContinuation(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	g := &Game{nativeMapAssets: assets, m: field, st: state, nativeMapVGA: make([]byte, 320*200)}
	called := false
	ui := &fakeSFXVoice{playing: true}
	voice := &fakeSFXVoice{playing: true, onClose: func() {
		if called || g.nativeMapVGA[4*320+4] != 3 {
			t.Error("physical stop did not follow map composition and precede continuation")
		}
	}}
	g.sfxVoices = []sfxVoice{ui}
	g.nativePhysicalVoice = voice
	g.atk = &atkAnim{nativeScene: &nativePhysicalScene{}, after: func() {
		called = true
		if g.nativeMapVGA[4*320+4] != 3 {
			t.Error("normal physical return continued before the ordinary map composer")
		}
		if !voice.closed || g.nativePhysicalVoice != nil || ui.closed {
			t.Error("continuation preceded physical stop or lost independent UI voice")
		}
	}}
	g.finishAttackPresentation()
	if !called || g.loadErr != "" {
		t.Fatalf("continuation=%v error=%q", called, g.loadErr)
	}
}

func TestNativePhysicalReturnSkipsSteadyPaletteCycle(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	frozen := time.Unix(1000, 0)
	before := bytes.Repeat([]byte{7}, 256*3)
	g := &Game{nativeMapAssets: assets, m: field, st: state,
		nativeMapDAC: append([]byte(nil), before...), nativeFDOTHERPalettePhase: 3,
		nativeMapFrozenNow: func() time.Time { return frozen }}
	if !g.nativeMapClock.Seed(10, frozen) {
		t.Fatal("BIOS seed")
	}
	continued := false
	g.atk = &atkAnim{nativeScene: &nativePhysicalScene{}, after: func() { continued = true }}
	g.finishAttackPresentation()
	if !continued || g.loadErr != "" || g.st.NativeMapCycleState.Moving != 1 {
		t.Fatalf("map return incomplete: continued=%v error=%q moving=%d", continued, g.loadErr, g.st.NativeMapCycleState.Moving)
	}
	if !bytes.Equal(g.nativeMapDAC, before) || g.nativeFDOTHERPalettePhase != 3 || g.nativeFDOTHERPaletteTick != 0 {
		t.Fatalf("11CAC(1) changed steady palette: phase=%d tick=%d", g.nativeFDOTHERPalettePhase, g.nativeFDOTHERPaletteTick)
	}
	// 同一BIOS tick的普通11CAC(0)仍須通過4DFCC gate。
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(g.nativeMapDAC, before) || g.nativeFDOTHERPalettePhase != 4 || g.nativeFDOTHERPaletteTick != 10 {
		t.Fatalf("ordinary redraw lost palette cycle: phase=%d tick=%d", g.nativeFDOTHERPalettePhase, g.nativeFDOTHERPaletteTick)
	}
}

func TestNativePhysicalMapReturnPreflightIsAtomic(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	g := &Game{nativeMapAssets: assets, m: field, st: state,
		nativeMapWork: bytes.Repeat([]byte{55}, indexedmap.NativeUnitPresentWorkSize),
		nativeMapVGA:  bytes.Repeat([]byte{77}, indexedmap.NativeMapVGASize),
		atk:           &atkAnim{nativeScene: &nativePhysicalScene{}}}
	before, unit, clock := *state, *state.Units[0], g.nativeMapClock
	if err := g.preflightNativePhysicalMapReturn(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, *state) || !reflect.DeepEqual(unit, *state.Units[0]) || clock != g.nativeMapClock || g.nativeMapDAC != nil ||
		!bytes.Equal(g.nativeMapWork, bytes.Repeat([]byte{55}, len(g.nativeMapWork))) || !bytes.Equal(g.nativeMapVGA, bytes.Repeat([]byte{77}, len(g.nativeMapVGA))) {
		t.Fatal("preflight published candidate state, clock or buffers")
	}
	assets.Frames = indexedmap.NativeMapHUDFrames{}
	if err := g.preflightNativePhysicalMapReturn(); err == nil {
		t.Fatal("declared incomplete map bundle was silently accepted")
	}
	if !reflect.DeepEqual(before, *state) || clock != g.nativeMapClock {
		t.Fatal("rejected preflight changed live state")
	}
}

func TestNativePhysicalMapReturnFailureSuppressesContinuation(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	assets.Frames = indexedmap.NativeMapHUDFrames{}
	voice := &fakeSFXVoice{playing: true}
	g := &Game{nativeMapAssets: assets, m: field, st: state, nativePhysicalVoice: voice}
	called := false
	g.atk = &atkAnim{nativeScene: &nativePhysicalScene{}, after: func() { called = true }}
	g.finishAttackPresentation()
	if called || !voice.closed || g.nativePhysicalVoice != nil || g.atk != nil || g.loadErr == "" {
		t.Fatalf("failed return continued=%v closed=%v attack=%v error=%q", called, voice.closed, g.atk, g.loadErr)
	}
}

func TestNativePhysicalSoundShutdownAndCompatibilityReturn(t *testing.T) {
	physical, ui, title := &fakeSFXVoice{playing: true}, &fakeSFXVoice{playing: true}, &fakeSFXVoice{playing: true}
	g := &Game{nativePhysicalVoice: physical, titleANI1Voice: title, sfxVoices: []sfxVoice{ui}}
	called := false
	g.atk = &atkAnim{nativeScene: &nativePhysicalScene{}, after: func() { called = true }}
	g.finishAttackPresentation()
	if !called || !physical.closed || ui.closed || title.closed {
		t.Fatal("nil map bundle return did not retain compatibility and independent voices")
	}
	physical = &fakeSFXVoice{playing: true}
	g.nativePhysicalVoice = physical
	g.closeAudioPlayers()
	if !physical.closed || !ui.closed || !title.closed || g.nativePhysicalVoice != nil {
		t.Fatal("shutdown retained a player")
	}
}

func TestNativePhysicalMapReturnMissingSourceStopsBeforeSettlement(t *testing.T) {
	requirePhysicalScenePack(t)
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			g, actor, target := physicalSceneTestGame(t)
			if err := g.ensureNativeAttackPresentation(actor.BattleFig, target.BattleFig); err != nil {
				t.Fatal(err)
			}
			g.nativeMapAssets = &nativeMapAssets{}
			beforeActor, beforeTarget := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "map return preflight") || g.atk != nil || g.nativeRNGState != 17791 ||
				!reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTarget, *target) {
				t.Fatalf("rejected map source changed attack transaction: %s", g.loadErr)
			}
			if got, want := g.rng.Int63(), rand.New(rand.NewSource(73)).Int63(); got != want {
				t.Fatal("rejected map source consumed legacy RNG")
			}
		})
	}
}
