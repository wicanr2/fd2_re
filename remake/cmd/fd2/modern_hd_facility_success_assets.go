package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
)

type modernFacilitySuccessBank struct {
	pictures          []*image.RGBA
	digests           [][32]byte
	frames            []*modernStoryPortraitFrame
	native            []modernLMIHDDescriptor
	baseFile, baseSHA string
	scene             *modernFacilityScene
	sceneImage        image.Image
}

func facilitySuccessShape(resource int) (count, width, height, offset int, roi image.Rectangle) {
	switch resource {
	case 12:
		return 5, 25, 17, 0x10bb, image.Rect(253, 181, 385, 264)
	case 29:
		return 1, 16, 19, 0x0f63, image.Rect(278, 165, 397, 239)
	case 63:
		return 7, 65, 31, 0x0e3c, image.Rect(254, 202, 390, 293)
	case 14:
		return 9, 3, 6, 0x0576, image.Rect(156, 180, 192, 227)
	}
	return
}

// Catalog declarations bind the original indexed entries, not their PNG colors.
func loadModernFacilitySuccessBank(set *modernStoryPortraitSet, a modernThemeAsset, root string) error {
	count, w, h, offset, _ := facilitySuccessShape(a.SourceResource)
	if count == 0 || a.Role != "facility_success_portrait_hd_set" || (a.Status != "runtime_candidate" && a.Status != "runtime_ready") ||
		a.ConsumerContract != "native_facility_success_portrait_hd6_v1" || a.SourceArchive != "FDOTHER.DAT" ||
		a.SourceMetadataSHA256 != modernFacilityMetadata[a.SourceResource] || a.Width != 480 || a.Height != 480 ||
		a.AlphaContract != "opaque" || a.CyclePolicy != "native_timeline" || a.LayoutPolicy != "native_portrait_patch_hd6" ||
		a.FrameCount != count || len(a.Files) != count || len(a.FrameSHA256) != count || len(a.NativeLMIFrames) != count ||
		len(a.SourceFiles) < 2 || len(a.SourceFiles) > 3 || len(a.SourceSHA256) != len(a.SourceFiles) || a.MasterFile != a.SourceFiles[1] ||
		len(a.NativeSourceRect) != 4 || a.NativeSourceRect[0] != offset%320 || a.NativeSourceRect[1] != offset/320 || a.NativeSourceRect[2] != 80 || a.NativeSourceRect[3] != 80 {
		return fmt.Errorf("modern facility success %d: invalid contract", a.SourceResource)
	}
	if set.facilitySuccess == nil {
		set.facilitySuccess = make(map[int]*modernFacilitySuccessBank)
	}
	if set.facilitySuccess[a.SourceResource] != nil {
		return fmt.Errorf("modern facility success: duplicate resource")
	}
	b := &modernFacilitySuccessBank{native: append([]modernLMIHDDescriptor(nil), a.NativeLMIFrames...), baseFile: a.SourceFiles[0], baseSHA: a.SourceSHA256[0]}
	seen := map[string]bool{}
	for i, name := range a.Files {
		d := a.NativeLMIFrames[i]
		if d.Index != 23+i || d.Width != w || d.Height != h || len(d.PixelsSHA256) != 64 || filepath.Base(name) != name || filepath.Ext(name) != ".png" || seen[name] {
			return fmt.Errorf("modern facility success: invalid source or path")
		}
		seen[name] = true
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != a.FrameSHA256[i] {
			return fmt.Errorf("modern facility success: sha256 mismatch")
		}
		im, format, err := image.Decode(bytes.NewReader(raw))
		if err != nil || format != "png" || im.Bounds() != image.Rect(0, 0, 480, 480) {
			return fmt.Errorf("modern facility success: invalid geometry")
		}
		for y := 0; y < 480; y++ {
			for x := 0; x < 480; x++ {
				_, _, _, alpha := im.At(x, y).RGBA()
				if alpha != 65535 {
					return fmt.Errorf("modern facility success: nonopaque pixel")
				}
			}
		}
		picture := image.NewRGBA(im.Bounds())
		draw.Draw(picture, picture.Bounds(), im, image.Point{}, draw.Src)
		b.pictures = append(b.pictures, picture)
		b.digests = append(b.digests, sha256.Sum256(picture.Pix))
		b.frames = append(b.frames, &modernStoryPortraitFrame{hdImage: picture})
	}
	set.facilitySuccess[a.SourceResource] = b
	set.hasHD = true
	return nil
}

func validateModernFacilitySuccessBanks(set *modernStoryPortraitSet) error {
	if len(set.facilitySuccess) == 0 {
		return nil
	}
	if len(set.facilitySuccess) != 4 {
		return fmt.Errorf("modern facility success: incomplete four-resource group")
	}
	for _, resource := range []int{12, 29, 63, 14} {
		b, s := set.facilitySuccess[resource], set.facilityScenes[resource]
		if b == nil || s == nil || s.file != b.baseFile || s.sha256 != b.baseSHA {
			return fmt.Errorf("modern facility success: base scene differs")
		}
		_, _, _, offset, roi := facilitySuccessShape(resource)
		origin := image.Pt(offset%320*6, offset/320*6)
		for _, f := range b.frames {
			for y := 0; y < 480; y++ {
				for x := 0; x < 480; x++ {
					if image.Pt(x, y).In(roi) {
						continue
					}
					if color.RGBAModel.Convert(f.hdImage.At(x, y)) != color.RGBAModel.Convert(s.image.At(origin.X+x, origin.Y+y)) {
						return fmt.Errorf("modern facility success: change escaped edit region")
					}
				}
			}
		}
		b.scene, b.sceneImage = s, s.image
	}
	return nil
}
