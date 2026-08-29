package musiccatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repositoryAssets(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller path unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "assets"))
}

func requireRenderBundle(t *testing.T) string {
	t.Helper()
	assets := repositoryAssets(t)
	if _, err := os.Stat(filepath.Join(assets, "music_fm", "FDMUS_001.ogg")); err != nil {
		t.Skipf("分離音樂 render 未安裝：%v", err)
	}
	return assets
}

func TestLoadValidatesAllThirtySeparatedRenders(t *testing.T) {
	catalog, err := Load(requireRenderBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"fm", "mt32"} {
		for _, track := range []string{"FDMUS_004", "FDMUS_018"} {
			path, err := catalog.Resolve(profile, track)
			if err != nil {
				t.Fatal(err)
			}
			if info, err := os.Stat(path); err != nil || info.Size() == 0 {
				t.Fatalf("%s/%s path=%q info=%v err=%v", profile, track, path, info, err)
			}
		}
	}
}

func TestLoadFailsClosedOnCatalogIdentityMutation(t *testing.T) {
	assets := repositoryAssets(t)
	raw, err := os.ReadFile(filepath.Join(assets, "music_catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	source := document["source"].(map[string]any)
	source["sha256"] = "0000000000000000000000000000000000000000000000000000000000000000"
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "music_catalog.json"), mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("mutated FDMUS identity was accepted")
	}
}

func TestLoadFailsClosedOnRenderHashMutation(t *testing.T) {
	assets := requireRenderBundle(t)
	raw, err := os.ReadFile(filepath.Join(assets, "music_catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	tracks := document["tracks"].([]any)
	first := tracks[0].(map[string]any)
	renders := first["renders"].(map[string]any)
	fm := renders["fm"].(map[string]any)
	fm["sha256"] = "0000000000000000000000000000000000000000000000000000000000000000"
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "music_catalog.json"), mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"music_fm", "music_mt32"} {
		if err := os.Symlink(filepath.Join(assets, directory), filepath.Join(root, directory)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Load(root); err == nil {
		t.Fatal("mutated OGG hash was accepted")
	}
}

func TestResolveRejectsUnknownProfileAndTrack(t *testing.T) {
	catalog, err := Load(requireRenderBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Resolve("legacy", "FDMUS_004"); err == nil {
		t.Fatal("legacy profile was accepted")
	}
	if _, err := catalog.Resolve("fm", "FDMUS_999"); err == nil {
		t.Fatal("unknown track was accepted")
	}
}
