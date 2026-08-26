package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func useFakeExecutableDir(t *testing.T, dir string) {
	t.Helper()
	oldDir, oldLooked := exeDirCached, exeDirLooked
	exeDirCached, exeDirLooked = dir, true
	t.Cleanup(func() {
		exeDirCached, exeDirLooked = oldDir, oldLooked
	})
}

func useIsolatedUserData(t *testing.T) {
	t.Helper()
	old := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("APPDIR", "")
	t.Cleanup(func() { userDataDirCached = old })
}

func writeTestAsset(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssetPathFindsMacBundleResources(t *testing.T) {
	useIsolatedUserData(t)
	bundle := t.TempDir()
	macOS := filepath.Join(bundle, "FD2.app", "Contents", "MacOS")
	useFakeExecutableDir(t, macOS)
	want := filepath.Join(bundle, "FD2.app", "Contents", "Resources", "assets", "spells.json")
	writeTestAsset(t, want, "{}")

	if got := assetPath("assets/spells.json"); got != want {
		t.Fatalf("assetPath = %q, want bundle resource %q", got, want)
	}
}

func TestAssetGlobFindsOneMacBundleLayer(t *testing.T) {
	useIsolatedUserData(t)
	bundle := t.TempDir()
	macOS := filepath.Join(bundle, "FD2.app", "Contents", "MacOS")
	useFakeExecutableDir(t, macOS)
	resources := filepath.Join(bundle, "FD2.app", "Contents", "Resources")
	want := []string{
		filepath.Join(resources, "assets", "story", "ch01.json"),
		filepath.Join(resources, "assets", "story", "ch30.json"),
	}
	for _, path := range want {
		writeTestAsset(t, path, "{}")
	}

	if got := assetGlob("assets/story/*.json"); !reflect.DeepEqual(got, want) {
		t.Fatalf("assetGlob = %#v, want %#v", got, want)
	}
}

func TestPackageSelfCheckUsesMacBundleAndFailsClosed(t *testing.T) {
	useIsolatedUserData(t)
	bundle := t.TempDir()
	macOS := filepath.Join(bundle, "FD2.app", "Contents", "MacOS")
	useFakeExecutableDir(t, macOS)
	resources := filepath.Join(bundle, "FD2.app", "Contents", "Resources")
	for _, rel := range []string{
		"assets/scenarios/campaign_full.json",
		"assets/spells.json",
		"assets/story/ch01.json",
		"assets/story/ch30.json",
	} {
		writeTestAsset(t, filepath.Join(resources, rel), "{}")
	}
	if err := packageSelfCheck(); err != nil {
		t.Fatalf("packageSelfCheck: %v", err)
	}

	writeTestAsset(t, filepath.Join(resources, "assets/story/ch30.json"), "not json")
	if err := packageSelfCheck(); err == nil {
		t.Fatal("packageSelfCheck accepted invalid bundled JSON")
	}
}
