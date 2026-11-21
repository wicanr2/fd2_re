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
	"sync"

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

// 天空之鑰兩個圖庫載入時只核對每格 PNG 的路徑、SHA-256 與標頭尺寸，不解碼（#317 後續）。
// 畫格在第一次使用時才解碼，只保留最近用到的幾張；透明度與原生黑幕條件在每格第一次解碼時
// 檢查，位元組已由 SHA-256 釘住，結果不會變。沒通過時整個圖庫退回原版並提示一次。
type modernSkyKeyHDBank struct {
	asset  modernThemeAsset
	plan   modernSkyKeyPlan
	root   string
	offset int
	lazy   *modernSkyKeyDecoded // 按需解碼的狀態；複製 bank 時共用同一份
}

// modernSkyKeyCacheSize 是同時保留的已解碼畫格數。兩個消費端（第二十章演出與開場 ANI0）都是
// 依序播放，同一格會連續用好幾個 tick，換格時才需要解碼新的一張。
const modernSkyKeyCacheSize = 3

type modernSkyKeyDecoded struct {
	mu       sync.Mutex
	recent   []modernSkyKeyCached // 依使用先後排列，最舊的在前
	pixelSHA map[string]string    // PNG SHA-256 → 第一次解碼的像素 SHA-256；快取命中與重解都必須相同
	summary  map[string]modernSkyKeyCoverage
	verified []bool // 依畫格：延後的透明度／黑幕條件已通過
	rejected error
	notices  *modernThemeNotices
}

type modernSkyKeyCached struct {
	hash    string
	picture *image.RGBA
}

// modernSkyKeyCoverage 摘要一張解碼圖的 alpha 範圍與是否有非黑像素。
type modernSkyKeyCoverage struct {
	low, high uint8
	nonblack  bool
}

func (b *modernSkyKeyHDBank) decodedState() *modernSkyKeyDecoded {
	if b.lazy == nil {
		b.lazy = &modernSkyKeyDecoded{}
	}
	d := b.lazy
	if d.pixelSHA == nil {
		d.pixelSHA = map[string]string{}
	}
	if d.summary == nil {
		d.summary = map[string]modernSkyKeyCoverage{}
	}
	if len(d.verified) != len(b.asset.Files) {
		d.verified = make([]bool, len(b.asset.Files))
	}
	return d
}

// rejectedErr 回報延後檢查是否已讓這個圖庫退回原版。
func (b *modernSkyKeyHDBank) rejectedErr() error {
	if b == nil || b.lazy == nil {
		return nil
	}
	b.lazy.mu.Lock()
	defer b.lazy.mu.Unlock()
	return b.lazy.rejected
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
	b := &modernSkyKeyHDBank{asset: asset, root: root, lazy: &modernSkyKeyDecoded{notices: set.themeNotices()}}
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
		if _, err := b.frameBytes(i); err != nil {
			return err
		}
	}
	if asset.Width != maxW || asset.Height != maxH {
		return errors.New("modern sky key: bank bounds changed")
	}
	b.decodedState()
	if b.offset == 1 {
		set.skyKeyFD34 = b
	} else {
		set.skyKeyANI = b
	}
	set.hasHD = true
	return nil
}

// frameBytes 讀取一格 PNG，核對路徑、SHA-256 與標頭尺寸；不解碼。
func (b *modernSkyKeyHDBank) frameBytes(index int) ([]byte, error) {
	if b == nil {
		return nil, errors.New("modern sky key: bank absent")
	}
	i := index - b.offset
	if i < 0 || i >= len(b.asset.Files) || i >= len(b.asset.NativeFrames) || i >= len(b.asset.FrameSHA256) {
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
	c, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || c.Width != w || c.Height != h {
		return nil, errors.New("modern sky key: PNG geometry changed")
	}
	return raw, nil
}

// picture 回傳一格已驗證、不可修改的畫格；呼叫端要改像素前先複製。每次呼叫都重新讀檔核對
// SHA-256。畫格第一次使用時才解碼，最近用到的幾張留在快取，命中時核對像素 SHA-256 沒被改過。
// 延後的透明度／黑幕條件沒有通過時，整個圖庫退回原版：回傳的錯誤符合
// errModernHDDeferredRejected，呼叫端改用原版畫格，不讓交易失敗。
func (b *modernSkyKeyHDBank) picture(index int) (*image.RGBA, error) {
	raw, err := b.frameBytes(index)
	if err != nil {
		return nil, err
	}
	i := index - b.offset
	hash := b.asset.FrameSHA256[i]
	d := b.asset.NativeFrames[i]
	bounds := image.Rect(0, 0, d.Width*6, d.Height*6)
	s := b.decodedState()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rejected != nil {
		return nil, s.rejected
	}
	var out *image.RGBA
	for n, cached := range s.recent {
		if cached.hash != hash {
			continue
		}
		if cached.picture.Bounds() != bounds || skyKeySHA(cached.picture.Pix) != s.pixelSHA[hash] {
			return nil, errors.New("modern sky key: decoded frame changed")
		}
		out = cached.picture
		s.recent = append(append(s.recent[:n:n], s.recent[n+1:]...), cached)
		break
	}
	if out == nil {
		im, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, s.reject(fmt.Errorf("modern sky key: frame %d cannot be decoded: %w", index, err))
		}
		out = image.NewRGBA(bounds)
		draw.Draw(out, out.Bounds(), im, image.Point{}, draw.Src)
		pixels := skyKeySHA(out.Pix)
		if prior, ok := s.pixelSHA[hash]; ok && prior != pixels {
			return nil, errors.New("modern sky key: decoded frame changed")
		}
		s.pixelSHA[hash] = pixels
		if _, ok := s.summary[hash]; !ok {
			s.summary[hash] = skyKeyCoverageOf(out)
		}
		s.recent = append(s.recent, modernSkyKeyCached{hash: hash, picture: out})
		if len(s.recent) > modernSkyKeyCacheSize {
			s.recent = append(s.recent[:0:0], s.recent[len(s.recent)-modernSkyKeyCacheSize:]...)
		}
	}
	if !s.verified[i] {
		c := s.summary[hash]
		if b.offset == 1 && (c.low != 0 || c.high != 255) || b.offset == 0 && (c.low != 255 || c.high != 255 || b.plan.Frames[index].Blank == c.nonblack) {
			return nil, s.reject(fmt.Errorf("modern sky key: frame %d coverage or native black frame changed", index))
		}
		s.verified[i] = true
	}
	return out, nil
}

func skyKeyCoverageOf(picture *image.RGBA) modernSkyKeyCoverage {
	c := modernSkyKeyCoverage{low: 255}
	for n := 0; n < len(picture.Pix); n += 4 {
		a := picture.Pix[n+3]
		if a < c.low {
			c.low = a
		}
		if a > c.high {
			c.high = a
		}
		c.nonblack = c.nonblack || picture.Pix[n] != 0 || picture.Pix[n+1] != 0 || picture.Pix[n+2] != 0
	}
	return c
}

// reject 讓整個圖庫退回原版並提示一次；呼叫時已持有 s.mu。
func (s *modernSkyKeyDecoded) reject(cause error) error {
	s.rejected = fmt.Errorf("%w: %v", errModernHDDeferredRejected, cause)
	s.recent = nil
	s.notices.reject(cause)
	return s.rejected
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
	s := b.decodedState()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cached := range s.recent {
		if cached.picture == nil || skyKeySHA(cached.picture.Pix) != s.pixelSHA[cached.hash] {
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
	if set.skyKeyFD34.rejectedErr() != nil || set.skyKeyANI.rejectedErr() != nil {
		return nil, nil, nil // 延後檢查已退回原版：整段演出改用原版畫格
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
