package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNativeConfirmationPulseCurrentOracle(t *testing.T) {
	run := os.Getenv("FD2_CONFIRMATION_PULSE_ORIGINAL")
	if run == "" {
		t.Skip("需要#35現行原版相位追蹤")
	}
	metadata, err := os.ReadFile(filepath.Join(run, "runner.json"))
	var runner struct {
		Commit    string `json:"dosgolem_commit"`
		SHA       string `json:"original_fd2_exe_sha256"`
		Dirty     int    `json:"dosgolem_tracked_dirty_files"`
		Untracked int    `json:"dosgolem_untracked_files"`
		Locked    bool   `json:"lock_ally_hp"`
		Cleared   bool   `json:"force_enemy_clear_declared"`
	}
	if err != nil || json.Unmarshal(metadata, &runner) != nil || runner.Commit != "a01ff084f038ddd1d133480f08d32e3b2e6f98c0" || runner.SHA != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" || runner.Dirty != 0 || runner.Untracked != 0 || runner.Locked || runner.Cleared {
		t.Fatal("原版runner來源或注入狀態不符")
	}
	f, err := os.Open(filepath.Join(run, "eip-trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var g Game
	var pending, initialized bool
	readTick, latchTick := 0, 0
	var hasRead, hasLatch bool
	updates, negatives := 0, 0
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	for scan.Scan() {
		var row struct {
			EIP, EAX, EBX string
		}
		if err := json.Unmarshal(scan.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		var value uint32
		if _, err := fmt.Sscanf(row.EAX, "0x%x", &value); err != nil {
			t.Fatal(err)
		}
		switch row.EIP {
		case "0x19B21":
			readTick = int(int32(value))
			if readTick < -0x8000 || readTick > 0x7fff || hasRead {
				t.Fatal("原版比較read缺失或順序錯誤")
			}
			hasRead = true
		case "0x19B30":
			if pending {
				t.Fatal("原版writer缺少counter consumer")
			}
			if !hasRead {
				t.Fatal("原版BIOS輸入缺失")
			}
			delta := int(int32(value))
			if delta >= 0 && delta < 2 {
				t.Fatal("原版writer處於等待delta")
			}
			if initialized && readTick-g.nativeClassUILastTick != delta {
				t.Fatal("原版latch不連續")
			}
			updates++
			if delta < 0 {
				negatives++
			}
			pending = true
		case "0x19B51":
			latchTick = int(int32(value))
			if !pending || hasLatch || latchTick < -0x8000 || latchTick > 0x7fff {
				t.Fatal("原版保存read缺失或順序錯誤")
			}
			if initialized {
				g.stepNativeClassUIPulseTicks(readTick, latchTick)
			}
			hasLatch = true
		case "0x19D23":
			if pending && hasLatch && !initialized && value <= 3 {
				g.nativeClassUIPulse = int(value)
				g.nativeClassUILastTick = latchTick
				g.nativeClassUIHasTick, initialized = true, true
			}
			if !pending || !hasLatch || value > 3 || g.nativeClassUIPulse != int(value) {
				t.Fatalf("原版counter=%d 重製=%d", value, g.nativeClassUIPulse)
			}
		case "0x19D2E":
			var cell uint32
			if _, err := fmt.Sscanf(row.EBX, "0x%x", &cell); err != nil || !pending || value != uint32(g.nativeClassUIPulse/2) || cell != 48+value {
				t.Fatal("原版圖格consumer不符")
			}
			pending = false
			hasRead, hasLatch = false, false
		}
	}
	if scan.Err() != nil || updates != 11786 || negatives != 1 || pending {
		t.Fatal("原版追蹤數量或邊界不符", updates, negatives, pending, scan.Err())
	}
	t.Logf("1筆原版初態及11785次相位／圖格轉移一致，包含一次signed BIOS負差值")
}

// #35: compare only the four named historical UI points. Each owner starts
// through title LOAD with an existing constructed slot. The postbattle UI
// uses the next chapter slot; this does not rerun or certify either battle.
func TestNativeConfirmationPulseHistoricalPoints(t *testing.T) {
	root, out := os.Getenv("FD2_CONFIRMATION_PULSE_ROOT"), os.Getenv("FD2_CONFIRMATION_PULSE_OUT")
	if root == "" || out == "" {
		t.Skip("需要#35原版四點與固定槽")
	}
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	var report []map[string]any
	var gpuJobs []confirmationPulseGPUJob
	for _, point := range []struct {
		chapter, slotChapter, seq int
		run, kind                 string
	}{
		{4, 4, 38, "sample-r13", "departure_prompt"},
		{4, 5, 1105, "sample-r13", "town_enter"},
		{5, 5, 38, "sample-r5", "departure_prompt"},
		{5, 6, 1335, "sample-r5", "town_enter"},
	} {
		name := fmt.Sprintf("ch%02d-%s-%d", point.chapter, point.kind, point.seq)
		t.Run(name, func(t *testing.T) {
			slot := filepath.Join(root, fmt.Sprintf("parity-slot-ch%02d", point.slotChapter), "FD2.SAV")
			t.Setenv("FD2_NATIVE_SAVE", slot)
			g := loadGame()
			if g.loadErr != "" || !g.confirmTitleLoadSlot(0) {
				t.Fatal("正常LOAD失敗", g.loadErr)
			}
			r := &parityReplay{t: t, g: g, out: out}
			r.settleTown()
			town := g.camp.NodeID()
			if town != fmt.Sprintf("town_ch%02d", point.slotChapter) {
				t.Fatal("固定槽城鎮不符", town)
			}
			for g.campSel != 2 {
				if !g.moveNativeTownSelection(1) {
					t.Fatal("出口方向鍵失敗")
				}
			}
			r.enterTownOption(2)
			if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.nativePreparationPromptActive() {
				t.Fatal("正常提示沒有交出輸入權")
			}
			originalPath := filepath.Join(root, fmt.Sprintf("parity-slot-ch%02d", point.chapter), point.run, fmt.Sprintf("checkpoint-%04d.png", point.seq))
			original := townDepartureReadPNG(t, originalPath)
			saved := g.nativeClassUIPulse
			frameName, _ := r.frame(point.kind)
			if frameName == "" || g.nativeClassUIPulse != saved {
				t.Fatal("候選改變正式相位或沒有畫面")
			}
			var diffs []int
			var gpuJob confirmationPulseGPUJob
			gpuJob.g, gpuJob.town = g, town
			for phase := 0; phase < 2; phase++ {
				candidate := townDepartureReadPNG(t, filepath.Join(out, fmt.Sprintf("remake-0000-p%d.png", phase)))
				gpuJob.frames = append(gpuJob.frames, candidate)
				diff := 0
				for y := 0; y < 200; y++ {
					for x := 0; x < 320; x++ {
						if candidate.At(x, y) != original.At(x, y) {
							diff++
						}
					}
				}
				diffs = append(diffs, diff)
				f, err := os.Create(filepath.Join(out, fmt.Sprintf("%s-p%d.png", name, phase)))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, candidate); err != nil {
					t.Fatal(err)
				}
				f.Close()
			}
			if diffs[0] != 0 && diffs[1] != 0 {
				t.Fatalf("完整RGB差異=%v", diffs)
			}
			// The normal input owner must still cancel and restore the town.
			if os.Getenv("FD2_CONFIRMATION_PULSE_GPU") == "1" {
				gpuJobs = append(gpuJobs, gpuJob)
			} else if !g.handleNativePreparationInput(nativePreparationInput{escape: true}) || !pump(t, g, 600, func() bool { return g.camp.NodeID() == town }) {
				t.Fatal("正常ESC沒有返回城鎮")
			}
			report = append(report, map[string]any{"chapter": point.chapter, "seq": point.seq, "kind": point.kind, "original": originalPath, "slot": slot, "diff_pixels": diffs, "scope": "完整RGB合法兩相位候選；postbattle只比較排版，未重跑戰鬥"})
		})
	}
	if len(gpuJobs) != 0 {
		probe := &confirmationPulseGPUProbe{t: t, jobs: gpuJobs}
		if err := ebiten.RunGame(probe); err != nil {
			t.Fatal(err)
		}
		if probe.err != "" || probe.index != len(gpuJobs) || len(probe.diffs) != 8 {
			t.Fatal("GPU驗收", probe.err, probe.index, probe.diffs)
		}
		for _, diff := range probe.diffs {
			if diff != 0 {
				t.Fatal("GPU完整RGBA差異", probe.diffs)
			}
		}
		for i := range report {
			report[i]["gpu_rgba_diff"] = probe.diffs[i*2 : i*2+2]
		}
	}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "validation.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

type confirmationPulseGPUJob struct {
	g      *Game
	town   string
	frames []*image.Paletted
}

type confirmationPulseGPUProbe struct {
	t                     *testing.T
	jobs                  []confirmationPulseGPUJob
	index, phase, updates int
	drawn                 bool
	diffs                 []int
	err                   string
}

func (p *confirmationPulseGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *confirmationPulseGPUProbe) Update() error {
	p.updates++
	if p.updates > 1000 {
		p.err = "GPU逾時"
		return ebiten.Termination
	}
	if p.drawn {
		p.drawn = false
		p.phase++
		if p.phase == 2 {
			j := p.jobs[p.index]
			if !j.g.handleNativePreparationInput(nativePreparationInput{escape: true}) || !pump(p.t, j.g, 600, func() bool { return j.g.camp.NodeID() == j.town }) {
				p.err = "GPU後ESC失敗"
				return ebiten.Termination
			}
			p.index++
			p.phase = 0
		}
	}
	if p.index == len(p.jobs) {
		return ebiten.Termination
	}
	g := p.jobs[p.index].g
	if err := g.Update(); err != nil {
		return err
	}
	g.nativeClassUIPulse = p.phase * 2 // explicit finite legal phase, not production clock locking
	return nil
}
func (p *confirmationPulseGPUProbe) Draw(screen *ebiten.Image) {
	if p.index >= len(p.jobs) || p.drawn {
		return
	}
	j := p.jobs[p.index]
	j.g.Draw(screen)
	rgba := make([]byte, 640*400*4)
	screen.ReadPixels(rgba)
	diff := 0
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, g, b, a := j.frames[p.phase].At(x/2, y/2).RGBA()
			offset := (y*640 + x) * 4
			if rgba[offset] != byte(r>>8) || rgba[offset+1] != byte(g>>8) || rgba[offset+2] != byte(b>>8) || rgba[offset+3] != byte(a>>8) {
				diff++
			}
		}
	}
	p.diffs = append(p.diffs, diff)
	p.drawn = true
}
