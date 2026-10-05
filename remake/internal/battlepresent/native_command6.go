package battlepresent

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// #154：只識別固定 FDOTHER #32 的完整 typed bytes，不依檔名授權越界。
// isolation_spec 記錄與原始資源雜湊的對照；這不是原版 heap 的重製。
func command6Bank32Fingerprint(effect *figani.Animation) bool {
	if effect == nil || len(effect.Frames) != figani.NativeCommand6EffectFrameCount {
		return false
	}
	h := sha256.New()
	h.Write([]byte("fd2-command6-typed-v1\x00"))
	h.Write([]byte{effect.HeaderByte1, effect.HeaderByte2, effect.HeaderByte4})
	var word [4]byte
	binary.LittleEndian.PutUint32(word[:], uint32(len(effect.Frames)))
	h.Write(word[:])
	for _, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return false
		}
		for _, value := range []int{frame.X, frame.Y, frame.Width, frame.Height, frame.Delay} {
			if int(int32(value)) != value {
				return false
			}
			binary.LittleEndian.PutUint32(word[:], uint32(int32(value)))
			h.Write(word[:])
		}
		h.Write([]byte{frame.RawByte4, frame.RawByte5, frame.RawByte7})
		h.Write(frame.Pixels)
		h.Write(frame.Mask)
	}
	return fmt.Sprintf("%x", h.Sum(nil)) == "3ebd12dfb0cb4fec35e968551672d6c644e1475dc2ac8a3353b523e3d05005d3"
}

// 保留唯一已觀測的 -422 opaque 寫入於獨立前置列，其餘界限沿嚴格 blit。
func blitCommand6EffectWorkFrame(work, prefix []byte, effect *figani.Animation, layer figani.NativeCommand6Layer) error {
	frame := effect.Frames[layer.Frame]
	y0 := 30 + frame.Y + layer.Y
	if y0 >= 0 {
		return blitCommand0WorkFrame(work, frame, layer.X, layer.Y)
	}
	x0 := 160 + frame.X + layer.X
	if len(work) != nativeCommand0WorkStride*nativeCommand0WorkHeight || len(prefix) != nativeCommand0WorkStride ||
		layer.Mode != 5 || layer.Channel != 2 || layer.Frame != 9 || !layer.Secondary ||
		x0 != 141 || y0 != -1 || frame.Width != 143 || frame.Height != 111 || !command6Bank32Fingerprint(effect) {
		return fmt.Errorf("unreviewed command6 signed work frame (%d,%d %dx%d)", x0, y0, frame.Width, frame.Height)
	}
	negative := 0
	for x := 0; x < frame.Width; x++ {
		if frame.Mask[x] != 0 {
			if -nativeCommand0WorkStride+x0+x != -422 {
				return errors.New("unreviewed command6 prefix write")
			}
			negative++
		}
	}
	if negative != 1 {
		return errors.New("command6 prefix writer unavailable")
	}
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			at := y*frame.Width + x
			if frame.Mask[at] == 0 {
				continue
			}
			logical := (y0+y)*nativeCommand0WorkStride + x0 + x
			if logical < 0 {
				prefix[len(prefix)+logical] = frame.Pixels[at]
			} else {
				work[logical] = frame.Pixels[at]
			}
		}
	}
	return nil
}

// ComposeNativeCommand6TargetFrame draws one typed mode4→target→mode5 plan
// onto the native work surface, preserving caller actor and packed target.
// It never advances or publishes battle state.
func ComposeNativeCommand6TargetFrame(base []byte, actor, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand6TargetFrame, display figani.NativeCommand6TargetDisplayFrame) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || effect == nil || len(effect.Frames) != figani.NativeCommand6EffectFrameCount {
		return nil, errors.New("battlepresent: incomplete command6 target frame input")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	prefix := make([]byte, nativeCommand0WorkStride)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	draw := func(layers []figani.NativeCommand6Layer) error {
		for _, layer := range layers {
			if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
				return fmt.Errorf("battlepresent: command6 frame %d unavailable", layer.Frame)
			}
			if err := blitCommand6EffectWorkFrame(work, prefix, effect, layer); err != nil {
				return fmt.Errorf("battlepresent: command6 mode%d channel%d: %w", layer.Mode, layer.Channel, err)
			}
		}
		return nil
	}
	if err := draw(frame.Mode4); err != nil {
		return nil, err
	}

	if err := blitCommand0WorkFrame(work, actor, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command6 actor: %w", err)
	}
	if display.Shader != -1 {
		shade := display.Shader >> 8
		if display.Shader&0xff != 0xb0 || shade < 2 || shade > 8 {
			return nil, errors.New("battlepresent: command6 packed shader unavailable")
		}
		if target.Width <= 0 || target.Height <= 0 || len(target.Pixels) != target.Width*target.Height || len(target.Mask) != len(target.Pixels) {
			return nil, errors.New("battlepresent: command6 target malformed")
		}
		target.Pixels = append([]byte(nil), target.Pixels...)
		for i, value := range target.Pixels {
			target.Pixels[i] = byte((int(value)+shade)&7) + 0xb0
		}
	}
	if err := blitCommand0WorkFrame(work, target, display.OffsetX, display.OffsetY); err != nil {
		return nil, fmt.Errorf("battlepresent: command6 target: %w", err)
	}
	if err := draw(frame.Mode5); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}

// ComposeNativeCommand6OrbitFrame preserves the caller's first-mode,
// actor-effect, second-mode order for one front or tail iteration.
func ComposeNativeCommand6OrbitFrame(base []byte, actorEffect *figani.Animation, target *figani.Frame, effect *figani.Animation, frame figani.NativeCommand6OrbitFrame) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || actorEffect == nil || len(actorEffect.Frames) == 0 ||
		effect == nil || len(effect.Frames) != figani.NativeCommand6EffectFrameCount {
		return nil, errors.New("battlepresent: incomplete command6 orbit frame input")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	draw := func(layers []figani.NativeCommand6Layer) error {
		for _, layer := range layers {
			if layer.Frame != 4 {
				return fmt.Errorf("battlepresent: command6 orbit frame %d unavailable", layer.Frame)
			}
			if err := blitCommand0WorkFrame(work, effect.Frames[layer.Frame], layer.X, layer.Y); err != nil {
				return fmt.Errorf("battlepresent: command6 mode%d channel%d: %w", layer.Mode, layer.Channel, err)
			}
		}
		return nil
	}
	if err := draw(frame.First); err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command6 orbit actor: %w", err)
	}
	if frame.DrawTarget {
		if target == nil {
			return nil, errors.New("battlepresent: command6 tail target unavailable")
		}
		if err := blitCommand0WorkFrame(work, *target, 0, 0); err != nil {
			return nil, fmt.Errorf("battlepresent: command6 orbit target: %w", err)
		}
	}
	if err := draw(frame.Second); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}

// ComposeNativeCommand6TransitionFrame reproduces one command6 sub_2BA22
// frame: mode4, actor-effect last frame, displaced target idle, then mode5.
func ComposeNativeCommand6TransitionFrame(base []byte, actorEffect, targetIdle *figani.Animation, effect *figani.Animation, frame figani.NativeCommand6TransitionFrame) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || actorEffect == nil || len(actorEffect.Frames) == 0 ||
		targetIdle == nil || len(targetIdle.Frames) == 0 || effect == nil || len(effect.Frames) != figani.NativeCommand6EffectFrameCount {
		return nil, errors.New("battlepresent: incomplete command6 transition input")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	prefix := make([]byte, nativeCommand0WorkStride)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	draw := func(layers []figani.NativeCommand6Layer) error {
		for _, layer := range layers {
			if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
				return fmt.Errorf("battlepresent: command6 transition effect frame %d unavailable", layer.Frame)
			}
			if err := blitCommand6EffectWorkFrame(work, prefix, effect, layer); err != nil {
				return err
			}
		}
		return nil
	}
	if err := draw(frame.Mode4); err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command6 transition actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], frame.TargetOffsetX, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command6 transition target: %w", err)
	}
	if err := draw(frame.Mode5); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}
