package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"reflect"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernSkyKeyRow struct {
	Index       int    `json:"index"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	CanvasSHA   string `json:"native_canvas_sha256"`
	CanvasFFSHA string `json:"native_canvas_ff_sha256"`
	PixelSHA    string `json:"pixel_sha256"`
	DACSHA      string `json:"dac_sha256"`
	Colours     []byte `json:"source_colours"`
	Blank       bool   `json:"blank"`
}

type modernSkyKeyPlan struct {
	Schema      int               `json:"schema_version"`
	Kind        string            `json:"kind"`
	MetadataSHA string            `json:"source_metadata_sha256"`
	Frames      []modernSkyKeyRow `json:"frames"`
}

type modernSkyKeyHDBank struct {
	pictures   []*image.RGBA
	pictureSHA []string
	asset      modernThemeAsset
	plan       modernSkyKeyPlan
	root       string
	offset     int
}

func skyKeySHA(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func skyKeySafeFile(name string) bool {
	return name != "" && name == filepath.Base(name) && name != "." && name != ".."
}

func loadModernSkyKeyHDAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	if set == nil || asset.Role != "story_animation_hd_set" || (asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.AlphaContract != "coverage" || asset.LayoutPolicy != "native_frame_xy_hd6" ||
		asset.FrameCount != len(asset.Files) || asset.FrameCount != len(asset.FrameSHA256) || asset.FrameCount != len(asset.NativeFrames) ||
		len(asset.SourceFiles) == 0 || len(asset.SourceFiles) != len(asset.SourceSHA256) || !skyKeySafeFile(asset.MasterFile) {
		return errors.New("modern sky key: invalid bank contract")
	}
	b := &modernSkyKeyHDBank{asset: asset, root: root}
	metadata := ""
	kind := ""
	rows := 0
	switch {
	case asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource == 34 && asset.ConsumerContract == "sky_key_fdother34_rgba_hd6_v1" && asset.FrameCount == 100 && set.skyKeyFD34 == nil:
		b.offset = 1
		metadata = "animations/fdother_034_ch20_sky_key/bank.json"
		kind = "sky_key_fdother34_hd6"
		rows = 101
	case asset.SourceArchive == "ANI.DAT" && asset.SourceResource == 0 && asset.ConsumerContract == "sky_key_ani0_rgba_hd6_v1" && asset.FrameCount == 96 && set.skyKeyANI == nil:
		metadata = "animations/ANI_000/animation.json"
		kind = "sky_key_ani0_hd6"
		rows = 96
	default:
		return errors.New("modern sky key: unsupported or duplicate source")
	}
	raw, err := os.ReadFile(separatedAssetPath(metadata))
	if err != nil {
		return err
	}
	if skyKeySHA(raw) != asset.SourceMetadataSHA256 {
		return errors.New("modern sky key: source metadata changed")
	}
	master := false
	plans := 0
	seen := map[string]bool{}
	for i, name := range asset.SourceFiles {
		if !skyKeySafeFile(name) || seen[name] {
			return errors.New("modern sky key: unsafe or duplicate provenance")
		}
		seen[name] = true
		if filepath.Ext(name) != ".json" {
			master = master || name == asset.MasterFile
			if filepath.Ext(name) != ".png" {
				return errors.New("modern sky key: invalid provenance type")
			}
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if skyKeySHA(raw) != asset.SourceSHA256[i] {
			return errors.New("modern sky key: provenance SHA changed")
		}
		master = master || name == asset.MasterFile
		switch filepath.Ext(name) {
		case ".json":
			plans++
			if err := json.Unmarshal(raw, &b.plan); err != nil {
				return err
			}
		case ".png":
		default:
			return errors.New("modern sky key: invalid provenance type")
		}
	}
	if !master || plans != 1 || b.plan.Schema != 1 || b.plan.Kind != kind || b.plan.MetadataSHA != asset.SourceMetadataSHA256 || len(b.plan.Frames) != rows {
		return errors.New("modern sky key: source plan absent or changed")
	}
	pictures := make([]*image.RGBA, asset.FrameCount)
	pictureSHA := make([]string, asset.FrameCount)
	shared := make(map[string]*image.RGBA)
	maxW, maxH := 0, 0
	for i, row := range b.plan.Frames {
		if row.Index != i || len(row.Colours) == 0 || len(row.Colours) > 256 {
			return errors.New("modern sky key: invalid source row")
		}
		for k, c := range row.Colours {
			if k > 0 && row.Colours[k-1] >= c {
				return errors.New("modern sky key: colours not unique and sorted")
			}
		}
		if i < b.offset {
			continue
		}
		desc := asset.NativeFrames[i-b.offset]
		if b.offset == 1 {
			want := modernBattleHDDescriptor{X: row.X, Y: row.Y, Width: row.Width, Height: row.Height}
			if desc != want || row.X < 0 || row.Y < 0 || row.Width <= 0 || row.Height <= 0 || row.X+row.Width > 320 || row.Y+row.Height > 200 {
				return errors.New("modern sky key: native FDOTHER descriptor changed")
			}
		} else if desc != (modernBattleHDDescriptor{Width: 320, Height: 200}) {
			return errors.New("modern sky key: native ANI geometry changed")
		}
		if desc.Width*6 > maxW {
			maxW = desc.Width * 6
		}
		if desc.Height*6 > maxH {
			maxH = desc.Height * 6
		}
		picture, err := b.pictureWithSharedStorage(i, shared)
		if err != nil {
			return err
		}
		pictures[i-b.offset] = picture
		pictureSHA[i-b.offset] = skyKeySHA(picture.Pix)
	}
	if asset.Width != maxW || asset.Height != maxH {
		return errors.New("modern sky key: bank bounds changed")
	}
	b.pictures, b.pictureSHA = pictures, pictureSHA
	if b.offset == 1 {
		set.skyKeyFD34 = b
	} else {
		set.skyKeyANI = b
	}
	set.hasHD = true
	return nil
}

// Return a validated immutable frame. Cold decoding uses private storage and
// publishes only after the complete bank passes. Callers copy before editing.
func (b *modernSkyKeyHDBank) picture(index int) (*image.RGBA, error) {
	return b.pictureWithSharedStorage(index, nil)
}

// Only a cold loader owns shared; every path is still read and hashed, and
// every descriptor/coverage condition is checked before sharing is published.
func (b *modernSkyKeyHDBank) pictureWithSharedStorage(index int, shared map[string]*image.RGBA) (*image.RGBA, error) {
	if b == nil {
		return nil, errors.New("modern sky key: bank absent")
	}
	i := index - b.offset
	if b == nil || i < 0 || i >= len(b.asset.Files) || i >= len(b.asset.NativeFrames) || i >= len(b.asset.FrameSHA256) {
		return nil, errors.New("modern sky key: frame outside bank")
	}
	name := b.asset.Files[i]
	if !skyKeySafeFile(name) || filepath.Ext(name) != ".png" {
		return nil, errors.New("modern sky key: unsafe PNG path")
	}
	raw, err := os.ReadFile(filepath.Join(b.root, name))
	if err != nil {
		return nil, err
	}
	if skyKeySHA(raw) != b.asset.FrameSHA256[i] {
		return nil, errors.New("modern sky key: frame SHA changed")
	}
	d := b.asset.NativeFrames[i]
	w, h := d.Width*6, d.Height*6
	if w <= 0 || h <= 0 || w > 1920 || h > 1200 {
		return nil, errors.New("modern sky key: PNG geometry changed")
	}
	if len(b.pictures) > 0 {
		if i >= len(b.pictures) || i >= len(b.pictureSHA) || b.pictures[i] == nil || b.pictures[i].Bounds() != image.Rect(0, 0, w, h) || skyKeySHA(b.pictures[i].Pix) != b.pictureSHA[i] {
			return nil, errors.New("modern sky key: decoded frame changed")
		}
		return b.pictures[i], nil
	}
	out := shared[b.asset.FrameSHA256[i]]
	if out == nil {
		c, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || c.Width != w || c.Height != h {
			return nil, errors.New("modern sky key: PNG geometry changed")
		}
		im, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		out = image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(out, out.Bounds(), im, image.Point{}, draw.Src)
	} else if out.Bounds() != image.Rect(0, 0, w, h) {
		return nil, errors.New("modern sky key: PNG geometry changed")
	}
	low, high := uint8(255), uint8(0)
	nonblack := false
	for n := 0; n < len(out.Pix); n += 4 {
		a := out.Pix[n+3]
		if a < low {
			low = a
		}
		if a > high {
			high = a
		}
		nonblack = nonblack || out.Pix[n] != 0 || out.Pix[n+1] != 0 || out.Pix[n+2] != 0
	}
	if b.offset == 1 && (low != 0 || high != 255) || b.offset == 0 && (low != 255 || high != 255 || b.plan.Frames[index].Blank == nonblack) {
		return nil, errors.New("modern sky key: coverage or native black frame changed")
	}
	if shared != nil {
		shared[b.asset.FrameSHA256[i]] = out
	}
	return out, nil
}

func (b *modernSkyKeyHDBank) recheckFiles() error {
	if b == nil {
		return errors.New("modern sky key: bank absent")
	}
	for i, name := range b.asset.SourceFiles {
		if !skyKeySafeFile(name) || i >= len(b.asset.SourceSHA256) {
			return errors.New("modern sky key: invalid provenance")
		}
		if filepath.Ext(name) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(b.root, name))
		if err != nil {
			return err
		}
		if skyKeySHA(raw) != b.asset.SourceSHA256[i] {
			return errors.New("modern sky key: provenance changed")
		}
	}
	for i, name := range b.asset.Files {
		if !skyKeySafeFile(name) || i >= len(b.asset.FrameSHA256) {
			return errors.New("modern sky key: invalid frame path")
		}
		raw, err := os.ReadFile(filepath.Join(b.root, name))
		if err != nil {
			return err
		}
		if skyKeySHA(raw) != b.asset.FrameSHA256[i] {
			return errors.New("modern sky key: frame changed")
		}
	}
	if len(b.pictures) != len(b.asset.Files) || len(b.pictureSHA) != len(b.asset.Files) {
		return errors.New("modern sky key: decoded bank incomplete")
	}
	for i, picture := range b.pictures {
		d := b.asset.NativeFrames[i]
		if picture == nil || picture.Bounds() != image.Rect(0, 0, d.Width*6, d.Height*6) || skyKeySHA(picture.Pix) != b.pictureSHA[i] {
			return errors.New("modern sky key: decoded bank changed")
		}
	}
	return nil
}

func skyKeyUsed(pixels []byte) []byte {
	var seen [256]bool
	for _, c := range pixels {
		seen[c] = true
	}
	var out []byte
	for c, yes := range seen {
		if yes {
			out = append(out, byte(c))
		}
	}
	return out
}

func (b *modernSkyKeyHDBank) validateFDFrame(index int, f fdother.Frame) error {
	if b == nil || b.offset != 1 || index < 0 || index >= len(b.plan.Frames) {
		return errors.New("modern sky key: FDOTHER row absent")
	}
	r := b.plan.Frames[index]
	if r.Index != index || r.X != f.X || r.Y != f.Y || r.Width != f.Width || r.Height != f.Height {
		return errors.New("modern sky key: FDOTHER source geometry changed")
	}
	a, z := make([]byte, 64000), bytes.Repeat([]byte{255}, 64000)
	if err := f.Blit(a, 320, -1); err != nil {
		return err
	}
	if err := f.Blit(z, 320, -1); err != nil {
		return err
	}
	if skyKeySHA(a) != r.CanvasSHA || skyKeySHA(z) != r.CanvasFFSHA {
		return errors.New("modern sky key: FDOTHER source pixels or mask changed")
	}
	var used []byte
	for n, c := range a {
		if c == z[n] {
			used = append(used, c)
		}
	}
	if !bytes.Equal(skyKeyUsed(used), r.Colours) {
		return errors.New("modern sky key: FDOTHER source colours changed")
	}
	if index > 0 && b.asset.NativeFrames[index-1] != (modernBattleHDDescriptor{X: f.X, Y: f.Y, Width: f.Width, Height: f.Height}) {
		return errors.New("modern sky key: FDOTHER bank descriptor changed")
	}
	return nil
}

func (b *modernSkyKeyHDBank) validateANIFrame(index int, pixels, dac []byte) error {
	if b == nil || b.offset != 0 || index < 0 || index >= len(b.plan.Frames) || len(pixels) != 64000 || len(dac) != 768 {
		return errors.New("modern sky key: ANI row absent")
	}
	r := b.plan.Frames[index]
	if r.Index != index || skyKeySHA(pixels) != r.PixelSHA || skyKeySHA(dac) != r.DACSHA || !bytes.Equal(skyKeyUsed(pixels), r.Colours) || b.asset.NativeFrames[index] != (modernBattleHDDescriptor{Width: 320, Height: 200}) {
		return errors.New("modern sky key: ANI source changed")
	}
	return nil
}

func (g *Game) modernSkyKeyPair(set *modernStoryPortraitSet) (*modernSkyKeyHDBank, *modernSkyKeyHDBank, error) {
	if set == nil || set.skyKeyFD34 == nil && set.skyKeyANI == nil {
		return nil, nil, nil
	}
	if set.skyKeyFD34 == nil || set.skyKeyANI == nil || len(set.mapSpritesHD) != 125 {
		return nil, nil, errors.New("modern sky key: incomplete animation or HD world")
	}
	return set.skyKeyFD34, set.skyKeyANI, nil
}

func (g *Game) preflightModernSkyKey(set *modernStoryPortraitSet, frames []fdother.Frame, clip *afm.Clip) error {
	f, a, err := g.modernSkyKeyPair(set)
	if err != nil || f == nil {
		return err
	}
	if len(frames) != 101 || clip == nil || len(clip.IndexedFrames) != 96 || len(clip.Palettes) != 96 {
		return errors.New("modern sky key: native banks incomplete")
	}
	if err = f.recheckFiles(); err != nil {
		return err
	}
	if err = a.recheckFiles(); err != nil {
		return err
	}
	for i, frame := range frames {
		if err = f.validateFDFrame(i, frame); err != nil {
			return fmt.Errorf("FDOTHER34/%d: %w", i, err)
		}
	}
	for i, pixels := range clip.IndexedFrames {
		if err = a.validateANIFrame(i, pixels, clip.Palettes[i]); err != nil {
			return fmt.Errorf("ANI0/%d: %w", i, err)
		}
	}
	return nil
}

func skyKeyPaletteMatches(dac []byte, want interface{}) bool {
	palette, err := fdother.VGAPaletteFromDAC(dac)
	return err == nil && reflect.DeepEqual(palette, want)
}
