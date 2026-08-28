package ending

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func isTransparentTAI003(frame fdother.Frame) bool {
	if frame.Width != 10 || frame.Height != 3 || len(frame.Indexed) != 30 || len(frame.Mask) != 30 || len(frame.Pixels) != 0 {
		return false
	}
	for _, value := range frame.Mask {
		if value != 0 {
			return false
		}
	}
	return true
}

// RenderFigureFadePass executes one direct 0x29164 non-mirrored pass. The
// caller must obtain restore from the native FDOTHER#56 backdrop path; it is
// copied to the left 320-pixel viewport of the 640-stride work surface before
// the secondary FIGANI frame is placed at stage*10. The separated TAI#3 frame
// is validated as the native transparent no-op, not a visible platform.
func RenderFigureFadePass(c *IndexedCompositor, work, restore []byte, tai003 fdother.Frame, secondary figani.Frame, pass FigureFadePass) error {
	if c == nil || len(work) != Width*Height*2 || len(restore) != Bytes || !isTransparentTAI003(tai003) || pass.Stage < 0 || pass.Stage > 8 || pass.SourceOffset != pass.Stage*10 || pass.PaletteDelta != pass.Stage*6 {
		return fmt.Errorf("ending: invalid native figure fade pass")
	}
	if err := CopyRect(work, Width*2, restore, Width, Width, Height, 0); err != nil {
		return err
	}
	if err := secondary.BlitAtBase(work, Width*2, pass.SourceOffset); err != nil {
		return err
	}
	if err := CopyRect(c.VGA, Width, work, Width*2, Width, Height, 0); err != nil {
		return err
	}
	return c.SetBaselineDelta(0, 255, pass.PaletteDelta)
}

// RenderMirrorFigureFadePass executes one recovered unit[+6]==0 pass from
// 0x29164. Unlike RenderFigureFadePass, native 0x292ad expects the caller's
// 640-stride work surface to already contain the backdrop at +0x140; it
// renders the primary frame at +0x140-stage*10 and optionally renders the
// secondary frame at +0x140 when arg4==0. TAI#3 is only validated here: the
// native caller has already staged its transparent no-op before this callee.
func RenderMirrorFigureFadePass(c *IndexedCompositor, work []byte, tai003 fdother.Frame, primary, secondary figani.Frame, pass MirrorFigureFadePass) error {
	if c == nil || len(work) != Width*Height*2 || pass.Stage < 0 || pass.Stage > 8 || pass.PrimarySourceOffset != 0x140-pass.Stage*10 || pass.PaletteDelta != pass.Stage*6 {
		return fmt.Errorf("ending: invalid native mirror figure fade pass")
	}
	if pass.DrawPlatform && !isTransparentTAI003(tai003) {
		return fmt.Errorf("ending: invalid native mirror TAI#3")
	}
	// 0x292ad first presents the right 320-pixel viewport, then mutates that
	// same viewport before presenting it again.
	if err := CopyRect(c.VGA, Width, work[Width:], Width*2, Width, Height, 0); err != nil {
		return err
	}
	if pass.DrawSecondary {
		if err := secondary.BlitAtBase(work, Width*2, 0x140); err != nil {
			return err
		}
	}
	if err := primary.BlitAtBase(work, Width*2, pass.PrimarySourceOffset); err != nil {
		return err
	}
	if err := CopyRect(c.VGA, Width, work[Width:], Width*2, Width, Height, 0); err != nil {
		return err
	}
	return c.SetBaselineDelta(0, 255, pass.PaletteDelta)
}
