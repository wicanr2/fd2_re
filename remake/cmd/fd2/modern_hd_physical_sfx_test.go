package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModernPhysicalSFXRealCatalog(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_HD_AUDIO_CATALOG"), os.Getenv("FD2_TEST_HD_AUDIO_PACK")
	if catalog == "" || pack == "" {
		t.Skip("requires explicit complete HD sound catalog and legal source pack")
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	count := 0
	for _, asset := range doc.Assets {
		if asset.Role != modernPhysicalSFXRole {
			continue
		}
		if err := loadModernPhysicalSFX(set, asset, pack); err != nil {
			t.Fatal(err)
		}
		count++
		t.Logf("resource%d cues%v sha%s", asset.SourceResource, asset.CueIDs, asset.SHA256)
	}
	if count != 2 || validateModernPhysicalSFXGroup(set) != nil {
		t.Fatal("incomplete registered sound pair")
	}
	for key, pcm := range set.physicalSFX {
		if len(pcm) == 0 || len(pcm)%4 != 0 || bytes.Equal(pcm, make([]byte, len(pcm))) {
			t.Fatal("silent or invalid PCM", key)
		}
	}
}

func TestModernPhysicalSoundThemeSelection(t *testing.T) {
	original, modern := []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}
	g := &Game{}
	if got := g.modernPhysicalSound(49, 4, original); !bytes.Equal(got, original) {
		t.Fatal("classic sound changed")
	}
	g.modernStoryPortraits = &modernStoryPortraitSet{physicalSFX: map[[2]int][]byte{{49, 4}: modern}}
	if got := g.modernPhysicalSound(49, 4, original); !bytes.Equal(got, modern) {
		t.Fatal("HD impact was not selected")
	}
	for _, key := range [][2]int{{49, 0}, {49, 2}, {50, 4}, {53, 2}} {
		if got := g.modernPhysicalSound(key[0], key[1], original); !bytes.Equal(got, original) {
			t.Fatalf("unregistered or miss cue changed: %v", key)
		}
	}
	if got := g.modernPhysicalSound(49, 4, nil); got != nil {
		t.Fatal("missing original unexpectedly gained sound")
	}
	g.modernStoryPortraits = nil
	if got := g.modernPhysicalSound(49, 4, original); !bytes.Equal(got, original) {
		t.Fatal("F2 back to classic did not restore original")
	}
	if !bytes.Equal(original, []byte{1, 2, 3, 4}) || !bytes.Equal(modern, []byte{5, 6, 7, 8}) {
		t.Fatal("selection mutated PCM")
	}
}

func TestModernPhysicalSFXGroupCompleteness(t *testing.T) {
	set := &modernStoryPortraitSet{}
	if err := validateModernPhysicalSFXGroup(set); err != nil {
		t.Fatal("old catalogs must remain valid", err)
	}
	set.physicalSFX = make(map[[2]int][]byte)
	for _, key := range [][2]int{{49, 1}, {49, 3}, {49, 4}, {49, 5}} {
		set.physicalSFX[key] = []byte{1}
	}
	if validateModernPhysicalSFXGroup(set) == nil {
		t.Fatal("partial pair accepted")
	}
	set.physicalSFX[[2]int{53, 2}] = []byte{1}
	if err := validateModernPhysicalSFXGroup(set); err != nil {
		t.Fatal(err)
	}
	delete(set.physicalSFX, [2]int{49, 4})
	set.physicalSFX[[2]int{49, 0}] = []byte{1}
	if validateModernPhysicalSFXGroup(set) == nil {
		t.Fatal("wrong cue replaced required impact")
	}
}

func TestModernPhysicalSFXDecodedAssets(t *testing.T) {
	fixture := os.Getenv("FD2_HD_IMPACT_TEST_ROOT")
	if fixture == "" {
		t.Skip("requires the maintained HD impact generator's OGG fixture")
	}
	raw, err := os.ReadFile(filepath.Join(fixture, "axe-hd-crisp.ogg"))
	if err != nil {
		t.Fatal(err)
	}
	digest := func(raw []byte) string {
		h := sha256.Sum256(raw)
		return hex.EncodeToString(h[:])
	}
	root := t.TempDir()
	metadata := []byte(`{"source_resource":49,"test":"metadata binding only"}`)
	metaRoot := filepath.Join(root, "original", "sfx", "FDOTHER_049")
	if err := os.MkdirAll(metaRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaRoot, "resource.json"), metadata, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_ASSET_PACK", filepath.Join(root, "original"))
	asset := modernThemeAsset{Role: modernPhysicalSFXRole, Status: "runtime_candidate", File: "hit.ogg",
		SHA256: digest(raw), SourceArchive: "FDOTHER.DAT", SourceResource: 49,
		SourceMetadataSHA256: digest(metadata), CueIDs: []int{1, 3, 4, 5},
		ConsumerContract: "physical_impact_ogg_v1", SampleRate: 48000, Channels: 1}
	write := func(data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "hit.ogg"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(raw)
	set := &modernStoryPortraitSet{}
	if err := loadModernPhysicalSFX(set, asset, root); err != nil {
		t.Fatal(err)
	}
	pcm := set.physicalSFX[[2]int{49, 4}]
	if len(set.physicalSFX) != 4 || len(pcm) == 0 || len(pcm)%4 != 0 || len(pcm) > 44100*4*2 {
		t.Fatal("invalid decoded sound shape")
	}
	if loadModernPhysicalSFX(set, asset, root) == nil || !bytes.Equal(pcm, set.physicalSFX[[2]int{49, 4}]) {
		t.Fatal("duplicate admission was not atomic")
	}
	for _, name := range []string{"digest", "metadata", "path", "duplicate-cue", "miss-cue", "bank", "rate", "channels", "codec", "truncated"} {
		t.Run(name, func(t *testing.T) {
			candidate := asset
			candidate.CueIDs = append([]int(nil), asset.CueIDs...)
			write(raw)
			switch name {
			case "digest":
				candidate.SHA256 = digest([]byte("wrong"))
			case "metadata":
				candidate.SourceMetadataSHA256 = digest([]byte("wrong"))
			case "path":
				candidate.File = "../hit.ogg"
			case "duplicate-cue":
				candidate.CueIDs = []int{4, 4}
			case "miss-cue":
				candidate.CueIDs = []int{0}
			case "bank":
				candidate.SourceResource = 50
			case "rate":
				candidate.SampleRate = 44100
			case "channels":
				candidate.Channels = 2
			case "codec":
				bad := bytes.Repeat([]byte{0}, len(raw))
				candidate.SHA256 = digest(bad)
				write(bad)
			case "truncated":
				candidate.SHA256 = digest(raw[:40])
				write(raw[:40])
			}
			target := &modernStoryPortraitSet{}
			if loadModernPhysicalSFX(target, candidate, root) == nil || len(target.physicalSFX) != 0 {
				t.Fatal("bad asset was admitted or partially published")
			}
		})
	}
}
