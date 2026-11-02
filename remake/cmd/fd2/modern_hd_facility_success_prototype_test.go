package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestModernHDFacilitySuccessPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	out := os.Getenv("FD2_HD_FACILITY_SUCCESS_PROTOTYPE_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	sha := func(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
	save := func(name string, raw []byte) map[string]any {
		t.Helper()
		p := filepath.Join(out, name+".png")
		im := image.NewPaletted(image.Rect(0, 0, 320, 200), g.nativeClassUI.palette)
		copy(im.Pix, raw)
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, im); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		return map[string]any{"file": name + ".png", "indexed_sha256": sha(raw)}
	}
	records := []map[string]any{}
	frames := 0
	for _, variant := range []int{1, 3, 5} {
		shopDialogueTestState(g, variant, "success", 0, 0)
		a, portrait, portraitID, ok := g.nativeShopState()
		if !ok {
			t.Fatal("shop source absent")
		}
		source, ok := g.composeNativeShopBare()
		if !ok {
			t.Fatal("bare source absent")
		}
		before := append([]byte(nil), source...)
		animation, final, err := campaign.ComposeNativeShopPurchaseSuccessFrames(source, a, portrait, portraitID, variant)
		if err != nil {
			t.Fatal(err)
		}
		timeline, ok := g.nativeShopSuccessTimeline()
		if !ok {
			t.Fatal("shop timeline absent")
		}
		plan, err := campaign.PlanNativeShopPurchaseSuccess(variant)
		if err != nil {
			t.Fatal(err)
		}
		if len(animation) != plan.FrameCount || !bytes.Equal(before, source) {
			t.Fatal("source changed")
		}
		rows := []map[string]any{save(fmt.Sprintf("shop-%02d-base", a.ResourceID), source)}
		for i, raw := range animation {
			row := save(fmt.Sprintf("shop-%02d-step-%02d", a.ResourceID, i), raw)
			f := a.SuccessFrames[i]
			row["shape"] = []int{f.Width, f.Height}
			row["source_entry"] = 23 + i
			row["pixels_sha256"] = sha(f.Indexed)
			row["mask_sha256"] = sha(f.Mask)
			rows = append(rows, row)
			frames++
		}
		rows = append(rows, save(fmt.Sprintf("shop-%02d-final", a.ResourceID), final))
		total := time.Duration(0)
		for _, step := range timeline {
			total += step.duration
		}
		records = append(records, map[string]any{"owner": "shop", "variant": variant, "resource": a.ResourceID, "native_animation_count": len(animation), "timeline_steps": len(timeline), "timeline_duration_ns": int64(total), "effect_offset": plan.EffectOffset, "restore_portrait": plan.RestorePortraitMode0, "frames": rows})
	}
	facilityDialogueTestOwner(g, "church", "revive_confirm", 0, 0)
	source, ok := g.composeNativeChurchReviveQuestion()
	if !ok {
		t.Fatal("revive source absent")
	}
	animation, final, err := campaign.ComposeNativeChurchReviveSuccessFrames(source, g.nativeClassUI.reviveFX, g.nativeClassUI.portrait)
	if err != nil {
		t.Fatal(err)
	}
	if !g.beginNativeChurchReviveSuccess(nil) || g.nativeClassUIJob == nil {
		t.Fatal("revive timeline absent")
	}
	rows := []map[string]any{save("church-14-base", source)}
	for i, raw := range animation {
		row := save(fmt.Sprintf("church-14-step-%02d", i), raw)
		f := g.nativeClassUI.reviveFX[i]
		row["shape"] = []int{f.Width, f.Height}
		row["source_entry"] = 23 + i
		row["pixels_sha256"] = sha(f.Indexed)
		row["mask_sha256"] = sha(f.Mask)
		rows = append(rows, row)
		frames++
	}
	rows = append(rows, save("church-14-final", final))
	total := time.Duration(0)
	for _, step := range g.nativeClassUIJob.timeline {
		total += step.duration
	}
	records = append(records, map[string]any{"owner": "church", "variant": 0, "resource": 14, "native_animation_count": len(animation), "timeline_steps": len(g.nativeClassUIJob.timeline), "timeline_duration_ns": int64(total), "effect_offset": 32*320 + 147, "restore_portrait": true, "frames": rows})
	if frames != 22 {
		t.Fatal("unexpected actual success sources", frames)
	}
	result := map[string]any{"status": "PASS", "scope": "existing typed success producers; original palette exports for visual inspection; no new original oracle", "animated_frames": frames, "records": records}
	raw, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "prototype-report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("success resources=4 animated_frames=%d", frames)
}
