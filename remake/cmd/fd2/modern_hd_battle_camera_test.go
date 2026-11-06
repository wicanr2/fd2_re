package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestModernBattleCameraReleasesCompletedOwner(t *testing.T) {
	theme := &modernStoryPortraitSet{}
	owner := &nativePhysicalScene{}
	r := &modernHDBattleGPU{owner: owner, theme: theme, cameraOwner: owner,
		cameraTheme: theme, cameraActive: true, cameraStartFrame: 100}
	g := &Game{modernStoryPortraits: theme, modernHDBattleGPU: r}
	g.syncModernHDBattleOwner()
	if r.owner != nil || r.cameraOwner != nil || r.cameraTheme != nil || r.cameraActive || r.cameraStartFrame != 0 {
		t.Fatal("completed camera retained battle owner")
	}
}

func TestModernBattleCameraSourceAdmission(t *testing.T) {
	base := bytes.Repeat([]byte{7}, 64000)
	snapshot := append([]byte(nil), base...)
	sources := &modernHDBattleSources{Base: base, BaseScenery: []modernHDBattleSource{{Archive: "BG.DAT"}}}
	for _, at := range []image.Point{{171, 4}, {0, 154}} {
		frame := figani.Frame{X: at.X, Y: at.Y, Width: 149, Height: 42}
		sources.BaseScenery = append(sources.BaseScenery, modernHDBattleSource{Frame: frame, PanelText: &modernBattlePanelText{native: frame}})
	}
	clean, ready := modernBattleCameraBase(sources)
	if !ready || !bytes.Equal(base, snapshot) {
		t.Fatal("camera admission rejected or changed native base")
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			want := byte(7)
			if x >= 171 && y >= 4 && y < 46 || x < 149 && y >= 154 && y < 196 {
				want = 0
			}
			if clean[y*320+x] != want {
				t.Fatal("camera removed pixels outside fixed UI", x, y)
			}
		}
	}
	clip := image.Rect(0, 0, 320, 200)
	sources.BaseScenery[1].Clip = &clip
	if _, ready := modernBattleCameraBase(sources); ready {
		t.Fatal("partially clipped panel admitted")
	}
	sources.BaseScenery[1].Clip = nil
	sources.BaseScenery = sources.BaseScenery[1:]
	if _, ready := modernBattleCameraBase(sources); ready {
		t.Fatal("camera without proven scenery admitted")
	}
}

func TestModernBattleCameraOwnerClock(t *testing.T) {
	a, b := 1, 2
	r := &modernHDBattleGPU{owner: &a, theme: &modernStoryPortraitSet{}}
	if got := r.presentationZoom(100, 1.07); got != 1 {
		t.Fatal(got)
	}
	if got := r.presentationZoom(127, 1.07); math.Abs(got-1.035) > 1e-9 {
		t.Fatal(got)
	}
	if got := r.presentationZoom(154, 1.07); got != 1.07 {
		t.Fatal(got)
	}
	r.quality = hdTextureLight
	if got := r.presentationZoom(160, 1.07); got != 1.07 {
		t.Fatal("quality changed camera clock", got)
	}
	r.owner = &b
	if got := r.presentationZoom(160, 1.07); got != 1 {
		t.Fatal("new owner inherited zoom", got)
	}
}

func TestModernPhysicalImpactHoldIsLatched(t *testing.T) {
	newGame := func(modern bool, impact bool, cue int) (*Game, *nativePhysicalScene) {
		g := &Game{}
		if modern {
			g.modernStoryPortraits = &modernStoryPortraitSet{battlePresentation: &modernBattlePresentationProfile{SchemaVersion: 1, MaxZoom: 1.07, ImpactHoldFrames: 4}}
		}
		p := &nativePhysicalBodyPlayback{drawn: true, jobs: []nativePhysicalBodyJob{
			{present: &battlepresent.NativePhysicalBodyPresent{}, waitMillis: 20, cue: cue, hdImpact: impact},
			{present: &battlepresent.NativePhysicalBodyPresent{}, waitMillis: 1000, cue: -1},
		}}
		return g, &nativePhysicalScene{body: p}
	}
	g, scene := newGame(true, true, 4)
	if err := g.stepNativePhysicalBodyMillis(scene, 20); err != nil || scene.body.index != 0 || math.Abs(scene.body.hdExtraMillis-200.0/3) > 1e-9 {
		t.Fatal("HD impact was not held", err, scene.body.index, scene.body.hdExtraMillis)
	}
	g.modernStoryPortraits = nil
	if err := g.stepNativePhysicalBodyMillis(scene, 40); err != nil || scene.body.index != 0 {
		t.Fatal("F2 changed an already latched hold", err)
	}
	if err := g.stepNativePhysicalBodyMillis(scene, 30); err != nil || scene.body.index != 1 || scene.body.hdExtraMillis != 0 {
		t.Fatal("hold did not end exactly once", err, scene.body.index)
	}
	for _, tc := range []struct {
		modern, impact bool
		cue            int
	}{{false, true, 4}, {true, false, 4}, {true, true, 0}} {
		g, scene = newGame(tc.modern, tc.impact, tc.cue)
		if err := g.stepNativePhysicalBodyMillis(scene, 20); err != nil || scene.body.index != 1 {
			t.Fatal("classic/miss/non-impact acquired a hold", tc, err, scene.body.index)
		}
	}
	g, scene = newGame(true, true, 4)
	scene.body.drawn = false
	if err := g.stepNativePhysicalBodyMillis(scene, 100); err != nil || scene.body.elapsedMillis != 0 || scene.body.index != 0 {
		t.Fatal("impact skipped required Draw", err)
	}
}

func TestModernBattlePresentationProfileValidation(t *testing.T) {
	root := t.TempDir()
	valid := []byte(`{"schema_version":1,"max_zoom":1.07,"impact_hold_frames":4}`)
	for _, tc := range []struct {
		name     string
		raw      []byte
		accepted bool
	}{
		{"valid", valid, true},
		{"zoom", []byte(`{"schema_version":1,"max_zoom":1.2,"impact_hold_frames":4}`), false},
		{"extra", []byte(`{"schema_version":1,"max_zoom":1.07,"impact_hold_frames":4,"other":0}`), false},
		{"trailing", append(append([]byte(nil), valid...), []byte(`{}`)...), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(root, "profile.json"), tc.raw, 0644); err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(tc.raw)
			asset := modernThemeAsset{Role: modernBattlePresentationRole, Status: "runtime_candidate", File: "profile.json", SHA256: hex.EncodeToString(h[:]), ConsumerContract: "hd1_battle_presentation_v1"}
			set := &modernStoryPortraitSet{}
			err := loadModernBattlePresentation(set, asset, root)
			if (err == nil) != tc.accepted || (set.battlePresentation != nil) != tc.accepted {
				t.Fatal("profile admission", err)
			}
			if tc.accepted && loadModernBattlePresentation(set, asset, root) == nil {
				t.Fatal("duplicate profile admitted")
			}
		})
	}
}
