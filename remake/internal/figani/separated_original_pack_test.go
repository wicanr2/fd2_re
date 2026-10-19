package figani

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// 完整唯讀分離包的資料回歸，並非 GUI、原版執行或新增動畫完成證據。
// 與工具的固定 FIGANI 容器 SHA 核對合用，避免只信 manifest 的 PNG 雜湊。
func TestSeparatedOriginalPackAllFIGANIFrames(t *testing.T) {
	pack := os.Getenv("FD2_TEST_ORIGINAL_PACK")
	if pack == "" {
		t.Skip("requires an explicit legal original pack")
	}
	resources, frames := 0, 0
	for resource := 0; resource < 409; resource++ {
		root := filepath.Join(pack, "animations", fmt.Sprintf("FIGANI_%03d", resource))
		if _, err := os.Stat(filepath.Join(root, "animation.json")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(pack, "raw", "FIGANI", fmt.Sprintf("FIGANI_%03d.bin", resource)))
		if err != nil {
			t.Fatal(err)
		}
		original, err := Parse(raw)
		if err != nil {
			t.Fatalf("resource %d: %v", resource, err)
		}
		separated, err := LoadSeparatedResource(filepath.Join(pack, "animations"), resource)
		if err != nil {
			t.Fatalf("resource %d: %v", resource, err)
		}
		if !reflect.DeepEqual(original, separated) {
			t.Fatalf("resource %d: separated pixels/mask/descriptor differ from raw decoder", resource)
		}
		resources++
		frames += len(original.Frames)
	}
	if resources != 264 || frames != 2118 {
		t.Fatalf("fixed original coverage changed: %d resources / %d frames", resources, frames)
	}
	t.Logf("%d resources / %d frames: raw decoder equals separated pixels, masks and descriptors", resources, frames)
}

func TestSeparatedOriginalCompoundFDOTHERFrames(t *testing.T) {
	pack := os.Getenv("FD2_TEST_ORIGINAL_PACK")
	if pack == "" {
		t.Skip("requires an explicit legal original pack")
	}
	frames := 0
	for resource := 65; resource <= 68; resource++ {
		raw, err := os.ReadFile(filepath.Join(pack, "raw", "FDOTHER", fmt.Sprintf("FDOTHER_%03d.bin", resource)))
		if err != nil {
			t.Fatal(err)
		}
		original, err := Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		separated, err := LoadSeparatedArchiveResource(filepath.Join(pack, "animations"), "FDOTHER.DAT", resource)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, separated) {
			t.Fatalf("FDOTHER%d: raw and separated pixels/mask/descriptor differ", resource)
		}
		frames += len(original.Frames)
	}
	if frames != 37 {
		t.Fatalf("fixed compound source frame count differs: %d", frames)
	}
	t.Logf("4 FDOTHER resources / %d frames: original parser equals separated sources", frames)
}
