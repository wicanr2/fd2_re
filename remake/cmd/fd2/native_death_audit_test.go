package main

import (
	"image"
	"reflect"
	"testing"
)

func TestNativeDeathAuditUsesDrawWitnessInsteadOfNextOwnerFrame(t *testing.T) {
	job := &nativeCh28PostPresentJob{frame: 19, presentedFrame: 15, presentedHD: true, drawn: false}
	g := &Game{nativeCh28PostPresent: job, modernHDMapPresented: &modernHDMapFrame{Viewport: image.Rect(0, 0, 1872, 1152),
		Overlays: []modernHDMapOverlayDraw{{Entry: 70, Rect: image.Rect(300, 400, 546, 640)}}}}
	before := *job
	result := make(map[string]any)
	g.recordNativeDeathAuditState(result)
	if result["unit_death_phase"] != "overlay_a" || result["unit_death_frame"] != 19 || result["unit_death_presented_frame"] != 15 ||
		!reflect.DeepEqual(result["unit_death_viewport"], []int{0, 0, 1872, 1152}) || !reflect.DeepEqual(before, *job) {
		t.Fatal("audit guessed the presented phase from the next owner frame or changed its owner")
	}
	job.presentedHD = false
	result = make(map[string]any)
	g.recordNativeDeathAuditState(result)
	if _, exists := result["unit_death_layers"]; exists {
		t.Fatal("faithful frame reported stale HD sources")
	}
	job.presentedFrame = -1
	result = make(map[string]any)
	g.recordNativeDeathAuditState(result)
	if result["unit_death_phase"] != "pending" {
		t.Fatal("unpresented owner claimed a Draw")
	}
	g.nativeCh28PostPresent = nil
	result = make(map[string]any)
	g.recordNativeDeathAuditState(result)
	if result["unit_death_scene"] != false || len(result) != 1 {
		t.Fatal("inactive owner retained witness fields")
	}
}
