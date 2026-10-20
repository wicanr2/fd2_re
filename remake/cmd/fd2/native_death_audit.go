package main

import "fmt"

// Only presentation metadata is observed. Current owner.frame may already
// follow the last Draw, so the screenshot witness uses presentedFrame instead.
func (g *Game) recordNativeDeathAuditState(result map[string]any) {
	result["unit_death_scene"] = g.nativeCh28PostPresent != nil
	job := g.nativeCh28PostPresent
	if job == nil {
		return
	}
	result["unit_death_owner"] = fmt.Sprintf("%p", job)
	result["unit_death_kind"] = job.kind
	result["unit_death_frame"] = job.frame
	result["unit_death_drawn"] = job.drawn
	result["unit_death_presented_frame"] = job.presentedFrame
	result["unit_death_presented_hd"] = job.presentedHD
	phase := "pending"
	switch frame := job.presentedFrame; {
	case frame >= 0 && frame < 13:
		phase = "pose"
	case frame >= 13 && frame < 19:
		phase = "overlay_a"
	case frame >= 19 && frame < 25:
		phase = "overlay_b"
	case frame == 25:
		phase = "steady"
	}
	result["unit_death_phase"] = phase
	result["unit_death_targets"] = job.hdTargets
	if !job.presentedHD || job.presentedFrame < 0 || g.modernHDMapPresented == nil {
		return
	}
	frame := g.modernHDMapPresented
	clip := frame.Viewport
	if clip.Empty() {
		clip = modernHDMapViewport()
	}
	result["unit_death_viewport"] = []int{clip.Min.X, clip.Min.Y, clip.Max.X, clip.Max.Y}
	layers := make([]map[string]any, 0, len(frame.Overlays))
	for _, layer := range frame.Overlays {
		layers = append(layers, map[string]any{"entry": layer.Entry, "rect": []int{layer.Rect.Min.X, layer.Rect.Min.Y, layer.Rect.Max.X, layer.Rect.Max.Y}})
	}
	result["unit_death_layers"] = layers
}
