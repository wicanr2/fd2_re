package main

import (
	"errors"
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// The native producer records the visible input alongside each indexed frame.
// In particular A retains the last pose although its published state is already
// inactive. Reconstructing A from stateFrames would remove the body too early.
func (g *Game) prepareModernHDDeathFrames(job *nativeCh28PostPresentJob, set *modernStoryPortraitSet) ([]*modernHDMapFrame, error) {
	if job == nil || set == nil || set.mapOverlays == nil || len(job.hdInputs) != 26 ||
		len(job.workFrames) != 26 || len(job.vgaFrames) != 26 || g.nativeMapAssets == nil {
		return nil, errors.New("modern HD death: incomplete sources")
	}
	if err := g.validateModernHDDeathStageSource(job); err != nil {
		return nil, err
	}
	var stage *nativeCh23AdapterState
	if job.hdStage != nil {
		stage = job.hdStage.stage
	}
	bank := set.mapOverlays
	for index := 68; index <= 79; index++ {
		if index >= len(g.nativeMapAssets.CommandHealDigits) {
			return nil, errors.New("modern HD death: native LMI entry absent")
		}
		want := modernLMIHDDescription(index, g.nativeMapAssets.CommandHealDigits[index])
		picture := bank.frames[index]
		if bank.native[index] != want || picture == nil || picture.Bounds() != image.Rect(0, 0, want.Width*6, want.Height*6) {
			return nil, fmt.Errorf("modern HD death: entry %d differs from native source", index)
		}
	}
	view := *g
	view.modernStoryPortraits = set
	frames := make([]*modernHDMapFrame, 26)
	for i, input := range job.hdInputs {
		if i < 25 {
			input.Frame.RangeMode = 0
			input.Frame.SelectionOverlay = nil
			// The native terrain/object passes retain the prior auxiliary seed;
			// only the final steady compose invokes the ChapterAux writer again.
			input.Frame.ChapterAux = nil
		}
		seed := job.workFrames[i]
		if stage != nil {
			seed = append([]byte(nil), seed...)
			if err := job.hdStage.seed(seed); err != nil {
				return nil, err
			}
		}
		frame, err := view.prepareModernHDMapFrameStageLayout(input, seed, job.palette, true, -1, stage)
		if err != nil {
			return nil, fmt.Errorf("modern HD death frame %d: %w", i, err)
		}
		// Existing event copies start at VGA offset zero. The final steady
		// compose alone returns to (4,4). Preserve every uncopied VGA byte.
		background := image.NewRGBA(image.Rect(0, 0, 320, 200))
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if i < 25 && x < 312 && y < 192 {
					background.Set(x, y, frame.Background.At(x+4, y+4))
				} else if i == 25 && x >= 4 && x < 316 && y >= 4 && y < 196 {
					background.Set(x, y, frame.Background.At(x, y))
				} else {
					background.Set(x, y, job.palette[job.vgaFrames[i][y*320+x]])
				}
			}
		}
		frame.Background = background
		if i < 25 {
			frame.UI = image.NewRGBA(image.Rect(0, 0, 320, 200))
			frame.HUDUnit = nil
			frame.HUDPanel = nil
			frame.Viewport = image.Rect(0, 0, 312*6, 192*6)
			shift := image.Pt(-24, -24)
			for k := range frame.Units {
				frame.Units[k].Rect = frame.Units[k].Rect.Add(shift)
			}
			for k := range frame.Terrain {
				frame.Terrain[k].Rect = frame.Terrain[k].Rect.Add(shift)
			}
			for k := range frame.Foreground {
				frame.Foreground[k].Rect = frame.Foreground[k].Rect.Add(shift)
			}
		}
		if i >= 13 && i < 25 {
			first, last := 68, 68+i-13
			if i >= 19 {
				first, last = 74+i-19, 74+i-19
			}
			for entry := first; entry <= last; entry++ {
				for _, target := range job.hdTargets {
					x := (target.Origin%fdother.NativeCh28PostStride - fdother.NativeCh28PostViewBase%fdother.NativeCh28PostStride) * 6
					y := (target.Origin/fdother.NativeCh28PostStride - fdother.NativeCh28PostViewBase/fdother.NativeCh28PostStride) * 6
					picture := bank.frames[entry]
					frame.Overlays = append(frame.Overlays, modernHDMapOverlayDraw{Entry: entry, Image: picture,
						Rect: picture.Bounds().Add(image.Pt(x, y))})
				}
			}
		}
		frames[i] = frame
	}
	return frames, nil
}
