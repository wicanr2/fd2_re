package main

import (
	"errors"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 離線候選，不由 Game.Draw 呼叫。READY-CANDIDATE 契約見規格61。
// HD 圖像的邊界與原生 descriptor 綁定；不依墨跡裁切或重算人物位置。
func composeModernHDBattleBodyCandidate(base []byte, attack, idle figani.Frame,
	rawSide byte, present battlepresent.NativePhysicalBodyPresent, palette color.Palette,
	attackHD, idleHD image.Image) (*image.RGBA, error) {
	if len(base) != 320*200 || len(palette) != 256 ||
		(present.OpaqueFill != -1 && present.OpaqueFill != 33) {
		return nil, errors.New("modern HD battle candidate: invalid base, palette or fill")
	}
	for _, value := range palette {
		if value == nil {
			return nil, errors.New("modern HD battle candidate: incomplete palette")
		}
	}
	for _, frame := range []figani.Frame{attack, idle} {
		if frame.Width <= 0 || frame.Height <= 0 || frame.Width > 1024 || frame.Height > 1024 ||
			len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) ||
			frame.X < -32768 || frame.X > 32767 || frame.Y < -32768 || frame.Y > 32767 {
			return nil, errors.New("modern HD battle candidate: invalid native frame")
		}
	}
	// 原始物理演出只使用有界的 staging 位移；拒絕會溢位的候選輸入。
	if present.DX < -32768 || present.DX > 32767 || present.DY < -32768 || present.DY > 32767 {
		return nil, errors.New("modern HD battle candidate: invalid displacement")
	}
	attackPicture, err := modernHDBattleCandidatePicture(attack, palette, attackHD)
	if err != nil {
		return nil, err
	}
	idlePicture, err := modernHDBattleCandidatePicture(idle, palette, idleHD)
	if err != nil {
		return nil, err
	}
	if present.OpaqueFill == 33 {
		filled := image.NewRGBA(idlePicture.Bounds())
		draw.DrawMask(filled, filled.Bounds(), &image.Uniform{C: palette[33]}, image.Point{},
			idlePicture, image.Point{}, draw.Src)
		idlePicture = filled
	}
	out := image.NewRGBA(image.Rect(0, 0, modernHDWidth, modernHDHeight))
	background := modernHDBattleCandidateIndexed(320, 200, base, nil, palette)
	draw.Draw(out, out.Bounds(), background, image.Point{}, draw.Src)
	blit := func(picture image.Image, x, y int) {
		rect := picture.Bounds().Add(image.Pt(x*6, y*6))
		draw.Draw(out, rect, picture, image.Point{}, draw.Over)
	}
	attackFirst := (rawSide == 0 && attack.RawByte7&1 == 0) || (rawSide != 0 && attack.RawByte7&1 != 0)
	if attackFirst {
		blit(attackPicture, attack.X, attack.Y)
	}
	blit(idlePicture, idle.X+present.DX, idle.Y+present.DY)
	if !attackFirst {
		blit(attackPicture, attack.X, attack.Y)
	}
	return out, nil
}

func modernHDBattleCandidatePicture(frame figani.Frame, palette color.Palette, hd image.Image) (image.Image, error) {
	if hd != nil {
		if hd.Bounds() != image.Rect(0, 0, frame.Width*6, frame.Height*6) {
			return nil, errors.New("modern HD battle candidate: HD source disagrees with native geometry")
		}
		return hd, nil
	}
	return modernHDBattleCandidateIndexed(frame.Width, frame.Height, frame.Pixels, frame.Mask, palette), nil
}

func modernHDBattleCandidateIndexed(width, height int, pixels, mask []byte, palette color.Palette) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, width*6, height*6))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if mask != nil && mask[i] == 0 {
				continue
			}
			value := color.NRGBAModel.Convert(palette[pixels[i]]).(color.NRGBA)
			for py := 0; py < 6; py++ {
				row := out.Pix[(y*6+py)*out.Stride+x*24:]
				for px := 0; px < 6; px++ {
					copy(row[px*4:px*4+4], []byte{value.R, value.G, value.B, value.A})
				}
			}
		}
	}
	return out
}
