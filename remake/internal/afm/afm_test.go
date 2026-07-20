package afm

import (
	"os"
	"path/filepath"
	"testing"
)

// aniPath 找玩家自備的 ANI.DAT(org_game;gitignore)。缺檔則測試 skip。
func aniPath() string {
	cands := []string{
		"../../../org_game/炎龍騎士團/FLAME2/ANI.DAT",
		"../../../../org_game/炎龍騎士團/FLAME2/ANI.DAT",
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// 各資源預期幀數(doc39 反組譯+視覺驗證)。
var wantFrames = map[int]int{
	0: 96, // 金鎖(橢圓→正圓)
	1: 51, // FLAME DRAGON 標題「2」logo 成形
	5: 35, // 拔劍屠龍
}

func TestDecodeANI(t *testing.T) {
	p := aniPath()
	if p == "" {
		t.Skip("ANI.DAT 不存在(玩家自備素材),跳過")
	}
	for res, want := range wantFrames {
		clip, err := DecodeResource(p, res)
		if err != nil {
			t.Fatalf("資源 %d 解碼失敗: %v", res, err)
		}
		if len(clip.Frames) != want {
			t.Errorf("資源 %d 幀數 = %d,預期 %d", res, len(clip.Frames), want)
		}
		if len(clip.IndexedFrames) != want || len(clip.Palettes) != want || len(clip.IndexedFrames[0]) != frameBytes || len(clip.Palettes[0]) != 768 {
			t.Errorf("資源 %d indexed snapshots不完整: frames=%d indexed=%d palettes=%d", res, len(clip.Frames), len(clip.IndexedFrames), len(clip.Palettes))
		}
		b := clip.Frames[0].Bounds()
		if b.Dx() != scrW || b.Dy() != scrH {
			t.Errorf("資源 %d 幀尺寸 = %dx%d,預期 320x200", res, b.Dx(), b.Dy())
		}
	}
}

func TestContainerReject(t *testing.T) {
	if _, err := containerEntries([]byte("not a container")); err == nil {
		t.Error("非容器資料應被拒絕")
	}
}

func TestDecodeANIDump(t *testing.T) {
	// 除錯輔助:設 AFM_DUMP=<dir> 時把資源 5 首/末幀存 PNG 供人眼對照(非 CI)。
	if os.Getenv("AFM_DUMP") == "" {
		t.Skip("未設 AFM_DUMP")
	}
	p := aniPath()
	if p == "" {
		t.Skip("ANI.DAT 不存在")
	}
	clip, err := DecodeResource(p, 5)
	if err != nil {
		t.Fatal(err)
	}
	_ = filepath.Join // 佔位;實際 dump 由外部工具負責
	t.Logf("資源 5:%d 幀,title=%q", len(clip.Frames), clip.Title)
}
