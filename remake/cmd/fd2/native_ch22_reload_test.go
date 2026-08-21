package main

import (
	"bytes"
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
