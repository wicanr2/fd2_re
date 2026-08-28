package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func nativeCh22OriginalPaths(t *testing.T) string {
	t.Helper()
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	for _, name := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT"} {
		if !fileExists(filepath.Join(base, name)) {
			t.Skip("player-provided " + name + " is unavailable")
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(base, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(base, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ASSET_PACK", "../../generated-assets/fd2-original-b97caf22")
	return base
}

func ch22ResourceBeat(source, archive, owner string, resource int) campaign.Beat {
	return campaign.Beat{Op: "load_res", Source: source, ResourceID: &resource, ResourceArchive: archive, ResourceOwner: owner}
}

func TestNativeCh22ReloadCommitsOnlyAfterGridResetAndAuxStage(t *testing.T) {
	nativeCh22OriginalPaths(t)
	g := completeNative2189AGame(t)
	g.handlerChapter = 23
	beforeMap, beforeGrid := g.m, append([]byte(nil), g.st.NativeMapEventGrid...)
	beats := []campaign.Beat{
		ch22ResourceBeat("0x24a4b", "FDFIELD.DAT", "0x53a51", 69),
		ch22ResourceBeat("0x24a65", "FDSHAP.DAT", "0x53a5d", 46),
		ch22ResourceBeat("0x24a7f", "FDSHAP.DAT", "0x53a69", 47),
	}
	for _, beat := range beats {
		if err := g.stageNativeCh22Resource(beat); err != nil {
			t.Fatal(err)
		}
		if g.m != beforeMap || !bytes.Equal(g.st.NativeMapEventGrid, beforeGrid) {
			t.Fatal("candidate resource load changed live map before commit")
		}
	}
	if err := g.resetNativeCh22ReloadGrid(); err != nil {
		t.Fatal(err)
	}
	if g.m != beforeMap || !bytes.Equal(g.st.NativeMapEventGrid, beforeGrid) {
		t.Fatal("candidate 0x4DBFC reset changed live map before auxiliary preflight")
	}
	if err := g.prepareNativeCh22Aux(); err != nil {
		t.Fatal(err)
	}
	if g.nativeCh22Reload != nil || g.m == beforeMap || g.m.W != 41 || g.m.H != 37 ||
		g.nativeMapAssets.MapIndex != 23 || len(g.st.NativeMapEventGrid) != 6072 ||
		len(g.nativeCh23State.staging) != fdother.NativeCh23StageStride*fdother.NativeCh23StageHeight {
		t.Fatalf("committed map=%dx%d mapIndex=%d grid=%d aux=%d", g.m.W, g.m.H, g.nativeMapAssets.MapIndex, len(g.st.NativeMapEventGrid), len(g.nativeCh23State.staging))
	}
	for index := 0; index < g.m.W*g.m.H; index++ {
		offset := 4 + 4*index
		if g.st.NativeMapEventGrid[offset+1]&^byte(0x03) != 0 ||
			g.st.NativeMapEventGrid[offset+2]&^byte(0x1f) != 0 ||
			g.st.NativeMapEventGrid[offset+3] != 0xff {
			t.Fatalf("cell%d was not reset by 0x4DBFC", index)
		}
	}
}

func TestNativeCh22ReloadRejectsMissingSeparatedAuxBeforeCommit(t *testing.T) {
	nativeCh22OriginalPaths(t)
	g := completeNative2189AGame(t)
	g.handlerChapter = 23
	beforeMap, beforeGrid := g.m, append([]byte(nil), g.st.NativeMapEventGrid...)
	for _, beat := range []campaign.Beat{
		ch22ResourceBeat("0x24a4b", "FDFIELD.DAT", "0x53a51", 69),
		ch22ResourceBeat("0x24a65", "FDSHAP.DAT", "0x53a5d", 46),
		ch22ResourceBeat("0x24a7f", "FDSHAP.DAT", "0x53a69", 47),
	} {
		if err := g.stageNativeCh22Resource(beat); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.resetNativeCh22ReloadGrid(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	if err := g.prepareNativeCh22Aux(); err == nil {
		t.Fatal("missing separated FDOTHER #42 auxiliary stage was accepted")
	}
	if g.nativeCh22Reload == nil || g.m != beforeMap ||
		!bytes.Equal(g.st.NativeMapEventGrid, beforeGrid) || g.nativeCh23State != nil {
		t.Fatal("missing separated FDOTHER #42 partially committed the reload")
	}
}

func TestNativeCh22ReloadRejectsWrongArchiveOwnerWithoutMutation(t *testing.T) {
	nativeCh22OriginalPaths(t)
	g := completeNative2189AGame(t)
	beforeMap, beforeGrid := g.m, append([]byte(nil), g.st.NativeMapEventGrid...)
	bad := ch22ResourceBeat("0x24a4b", "FDSHAP.DAT", "0x53a51", 69)
	if err := g.stageNativeCh22Resource(bad); err == nil {
		t.Fatal("wrong archive was accepted")
	}
	if g.nativeCh22Reload != nil || g.m != beforeMap || !bytes.Equal(g.st.NativeMapEventGrid, beforeGrid) {
		t.Fatal("failed resource binding changed live state")
	}
}

func TestSeparatedMap23CompositionMatchesFDFIELDResource69(t *testing.T) {
	base := nativeCh22OriginalPaths(t)
	_, got, err := loadSeparatedFDFIELDComposition("assets/maps/map23")
	if err != nil {
		t.Fatal(err)
	}
	want, err := fdother.ReadResource(filepath.Join(base, "FDFIELD.DAT"), 69)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("separated map23 composition differs: got=%d want=%d", len(got), len(want))
	}
}

func TestNativeCh22ReloadDoesNotReadFDFIELDOrFDSHAPArchives(t *testing.T) {
	nativeCh22OriginalPaths(t)
	g := completeNative2189AGame(t)
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(t.TempDir(), "missing-FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(t.TempDir(), "missing-FDSHAP.DAT"))
	beat := ch22ResourceBeat("0x24a4b", "FDFIELD.DAT", "0x53a51", 69)
	if err := g.stageNativeCh22Resource(beat); err != nil {
		t.Fatal(err)
	}
	if g.nativeCh22Reload == nil || g.nativeCh22Reload.field.W != 41 || g.nativeCh22Reload.field.H != 37 {
		t.Fatal("separated map23 candidate was not staged")
	}
}

func TestSeparatedFDFIELDCompositionRejectsIncompleteOrOutOfRangeData(t *testing.T) {
	for _, test := range []struct {
		name  string
		field MapData
	}{
		{"missing event byte", MapData{W: 1, H: 1, Tiles: []int{0}, NativeTileBlitModes: []byte{0}}},
		{"tile outside uint16", MapData{W: 1, H: 1, Tiles: []int{0x10000}, NativeCompositionEventBytes: []byte{0}, NativeTileBlitModes: []byte{0}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			raw, err := json.Marshal(test.field)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "map.json"), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := loadSeparatedFDFIELDComposition(dir); err == nil {
				t.Fatal("malformed separated composition was accepted")
			}
		})
	}
}
