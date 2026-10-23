package main

import (
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Decode only the verified transparent FDOTHER blit. Two different initial
// destinations distinguish an opaque index0 from a transparent write omission.
func modernHDBattleSurfaceFrame(frame fdother.Frame) (figani.Frame, error) {
	if frame.Width <= 0 || frame.Height <= 0 || frame.Width > 320 || frame.Height > 200 {
		return figani.Frame{}, fmt.Errorf("modern HD battle: invalid native surface")
	}
	a := make([]byte, frame.Width*frame.Height)
	b := make([]byte, len(a))
	for i := range b {
		b[i] = 255
	}
	local := frame
	local.X, local.Y = 0, 0
	if err := local.Blit(a, frame.Width, -1); err != nil {
		return figani.Frame{}, err
	}
	if err := local.Blit(b, frame.Width, -1); err != nil {
		return figani.Frame{}, err
	}
	mask := make([]byte, len(a))
	for i := range a {
		switch {
		case a[i] == b[i]:
			mask[i] = 255
		case a[i] == 0 && b[i] == 255:
		default:
			return figani.Frame{}, fmt.Errorf("modern HD battle: native surface depends on destination")
		}
	}
	return figani.Frame{X: frame.X, Y: frame.Y, Width: frame.Width, Height: frame.Height, Pixels: a, Mask: mask}, nil
}

func nativePhysicalBackgroundPresents(in battlepresent.NativeCommand24BackgroundInputs, right bool) ([]battlepresent.NativeCommand24BackgroundPresent, error) {
	if right {
		return battlepresent.BuildNativePhysicalRightBackgroundPresents(in)
	}
	return battlepresent.BuildNativeCommand24BackgroundPresents(in)
}

// The CPU producer owns phase, viewport and BG write history. The optional HD
// consumer supplies clean bases and explicit FIGANI layers, never erased ink.
func modernHDBattleScrollSources(presents []battlepresent.NativeCommand24BackgroundPresent, source *modernHDBattleSources,
	target []byte, idle figani.Frame, idleResource int, layers [3]fdother.Frame) ([]*modernHDBattleSources, error) {
	return modernHDBattleScrollScenerySources(presents, source, &modernHDBattleSources{Base: target}, idle, idleResource, layers)
}

func modernHDBattleScrollScenerySources(presents []battlepresent.NativeCommand24BackgroundPresent, source, target *modernHDBattleSources,
	idle figani.Frame, idleResource int, layers [3]fdother.Frame) ([]*modernHDBattleSources, error) {
	if source == nil || target == nil || len(source.Base) != 64000 || len(target.Base) != 64000 || len(presents) == 0 {
		return nil, fmt.Errorf("modern HD battle: scroll bases unavailable")
	}
	var backgrounds [3]figani.Frame
	for i, layer := range layers {
		if layer.Width != 320 || layer.Height != 100 {
			return nil, fmt.Errorf("modern HD battle: scroll BG geometry differs")
		}
		layer.X, layer.Y = 0, 50
		layers[i] = layer
		var err error
		backgrounds[i], err = modernHDBattleSurfaceFrame(layer)
		if err != nil {
			return nil, err
		}
	}
	out := make([]*modernHDBattleSources, 0, len(presents))
	for _, present := range presents {
		if len(present.Pixels) != 64000 || (present.ContentX != 0 && present.ContentX != 320) || present.BGX != 320-present.ContentX ||
			present.ViewportX < 0 || present.ViewportX > 320 || len(present.BGLayers) == 0 || len(present.BGLayers) > 10 {
			return nil, fmt.Errorf("modern HD battle: scroll metadata differs")
		}
		base := source.Base
		if present.Incoming {
			base = target.Base
		}
		work := make([]byte, 640*200)
		for y := 0; y < 200; y++ {
			copy(work[y*640+present.ContentX:], base[y*320:(y+1)*320])
		}
		for _, index := range present.BGLayers {
			if index < 0 || index >= len(layers) {
				return nil, fmt.Errorf("modern HD battle: scroll BG index differs")
			}
			if err := layers[index].BlitAt(work, 640, present.BGX, -1); err != nil {
				return nil, err
			}
		}
		clean := make([]byte, 64000)
		for y := 0; y < 200; y++ {
			copy(clean[y*320:(y+1)*320], work[y*640+present.ViewportX:y*640+present.ViewportX+320])
		}
		value := &modernHDBattleSources{Base: clean}
		shift := present.ContentX - present.ViewportX
		appendShifted := func(layers []modernHDBattleSource) {
			for _, layer := range layers {
				clip := image.Rect(0, 0, 320, 200)
				if layer.Clip != nil {
					clip = clip.Intersect(*layer.Clip)
				}
				clip = clip.Add(image.Pt(shift, 0))
				layer.Clip = &clip
				layer.DX += shift
				value.Layers = append(value.Layers, layer)
			}
		}
		if present.Incoming {
			appendShifted(target.BaseScenery)
			// TargetIdle is blitted into the 640-wide work, not clipped to its
			// nominal half. The later BG writes can cover any crossing pixels.
			value.Layers = append(value.Layers, modernHDBattleSource{Frame: idle, Resource: idleResource, Index: 0, DX: shift, Fill: -1})
			if present.ContentX+idle.X+idle.Width > 640 {
				// BlitAtBase adds a linear base to each row. A descriptor wider
				// than the remaining work row wraps to x0 on the next row.
				value.Layers = append(value.Layers, modernHDBattleSource{Frame: idle, Resource: idleResource, Index: 0, DX: shift - 640, DY: 1, Fill: -1})
			}
		} else {
			appendShifted(source.BaseScenery)
			appendShifted(source.Layers)
		}
		// Replay the native mask writes after TargetIdle. Repeating them on
		// the already composed clean base is idempotent, including index0.
		for _, index := range present.BGLayers {
			value.Layers = append(value.Layers, modernHDBattleSource{Frame: backgrounds[index], Resource: -1, DX: present.BGX - present.ViewportX, Fill: -1})
		}
		out = append(out, value)
	}
	return out, nil
}
