package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	NativeTownWidth        = 320
	NativeTownHeight       = 200
	nativeTownLabelX       = 244
	nativeTownLabelY       = 162
	nativeTownTextOffset   = 168*NativeTownWidth + 252
	nativeTownViewportX    = 4
	nativeTownViewportY    = 4
	nativeTownViewportW    = 312
	nativeTownViewportH    = 192
	nativeTownTextBase     = 0x1ef
	nativeTownVariantCount = 3
)

var nativeTownBackgroundResources = [...]int{11, 61, 62}

var nativeTownSelectionX = [nativeTownVariantCount][6]int{
	{29, 41, 59, 154, 182, 10},
	{90, 33, 53, 148, 222, 196},
	{59, 10, 59, 130, 242, 136},
}

var nativeTownSelectionY = [nativeTownVariantCount][6]int{
	{46, 109, 163, 139, 65, 10},
	{30, 105, 163, 139, 85, 8},
	{26, 144, 163, 150, 31, 20},
}

// NativeTownAssets are the exact resources consumed by
// 0x2cd46..0x2d05a: three full-screen backgrounds, FDOTHER#10's opaque
// current-label panel, and FDICON's first three pulse sprites.
type NativeTownAssets struct {
	Backgrounds [nativeTownVariantCount][]byte
	Label       fdother.LMI1Entry
	Pulse       [3]fdicon.Sprite
}

func DecodeNativeTownAssetsArchive(
	fdotherPath, fdiconPath string,
) (*NativeTownAssets, error) {
	if fdotherPath == "" || fdiconPath == "" {
		return nil, errors.New("campaign: native town asset path is empty")
	}
	bank, err := fdicon.DecodeFile(filepath.Clean(fdiconPath))
	if err != nil {
		return nil, err
	}
	return decodeNativeTownAssets(fdotherPath, bank)
}

type separatedTownLabelDocument struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	AssetID       string `json:"asset_id"`
	Status        string `json:"status"`
	Evidence      string `json:"evidence"`
	Codec         string `json:"codec"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Frame         string `json:"frame"`
	Source        struct {
		File     string `json:"file"`
		Resource int    `json:"resource"`
		Size     int    `json:"size"`
		MD5      string `json:"md5"`
		SHA256   string `json:"sha256"`
		RawSize  int    `json:"raw_size"`
	} `json:"source"`
}

// LoadNativeTownAssets uses only separated FDOTHER surfaces/label and the
// separated FDICON bank. The archive adapter above remains source-oracle only.
func LoadNativeTownAssets(packRoot string) (*NativeTownAssets, error) {
	if packRoot == "" {
		return nil, errors.New("campaign: native town asset path is empty")
	}
	bank, err := fdicon.LoadSeparatedBank(filepath.Join(packRoot, "sprites", "fdicon"))
	if err != nil {
		return nil, err
	}
	out := &NativeTownAssets{}
	for variant, resource := range nativeTownBackgroundResources {
		frame, err := fdother.LoadSeparatedSingleFrame(filepath.Join(packRoot, "surfaces"), "FDOTHER.DAT", resource)
		if err != nil {
			return nil, fmt.Errorf("campaign: separated town background %d: %w", resource, err)
		}
		if frame.Width != NativeTownWidth || frame.Height != NativeTownHeight {
			return nil, fmt.Errorf("campaign: separated town background %d is not 320x200", resource)
		}
		background := make([]byte, NativeTownWidth*NativeTownHeight)
		if err := frame.Blit(background, NativeTownWidth, -1); err != nil {
			return nil, err
		}
		out.Backgrounds[variant] = background
	}
	dir := filepath.Join(packRoot, "ui", "fdother_010_town_label")
	raw, err := os.ReadFile(filepath.Join(dir, "resource.json"))
	if err != nil {
		return nil, err
	}
	var doc separatedTownLabelDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if doc.SchemaVersion != 1 || doc.Kind != "native_town_label" || doc.AssetID != "ui/FDOTHER_010/town_label" || doc.Status != "decoded" || doc.Evidence != "confirmed" || doc.Codec != "opaque_high_run" || doc.Width != 62 || doc.Height != 26 || doc.Frame != "frame.png" || doc.Source.File != "FDOTHER.DAT" || doc.Source.Resource != 10 || doc.Source.Size != separatedFDOTHERSize || doc.Source.MD5 != separatedFDOTHERMD5 || doc.Source.SHA256 != separatedFDOTHERSHA256 || doc.Source.RawSize <= 4 {
		return nil, errors.New("campaign: separated town label contract mismatch")
	}
	pixels, err := readSeparatedIndexed(filepath.Join(dir, doc.Frame), doc.Width, doc.Height)
	if err != nil {
		return nil, err
	}
	out.Label = fdother.LMI1Entry{Width: doc.Width, Height: doc.Height, Pixels: pixels}
	if bank == nil || len(bank.Sprites) != fdicon.SeparatedSpriteCount {
		return nil, errors.New("campaign: native town FDICON pulse is incomplete")
	}
	copy(out.Pulse[:], bank.Sprites[:len(out.Pulse)])
	return out, nil
}

func decodeNativeTownAssets(fdotherPath string, bank *fdicon.Bank) (*NativeTownAssets, error) {
	out := &NativeTownAssets{}
	for variant, resourceID := range nativeTownBackgroundResources {
		raw, err := fdother.ReadResource(fdotherPath, resourceID)
		if err != nil {
			return nil, err
		}
		frame, err := fdother.ParseSingleFrame(raw)
		if err != nil || frame.Width != NativeTownWidth ||
			frame.Height != NativeTownHeight {
			return nil, errors.New(
				"campaign: native town background is not 320x200",
			)
		}
		background := make([]byte, NativeTownWidth*NativeTownHeight)
		if err := frame.Blit(background, NativeTownWidth, -1); err != nil {
			return nil, err
		}
		out.Backgrounds[variant] = background
	}
	labelRaw, err := fdother.ReadResource(fdotherPath, 10)
	if err != nil {
		return nil, err
	}
	out.Label, err = fdother.ParseOpaqueRunCell(labelRaw)
	if err != nil || out.Label.Width != 62 || out.Label.Height != 26 {
		return nil, errors.New(
			"campaign: native town label panel is not 62x26",
		)
	}
	if bank == nil || len(bank.Sprites) != fdicon.SeparatedSpriteCount {
		return nil, errors.New("campaign: native town FDICON pulse is incomplete")
	}
	copy(out.Pulse[:], bank.Sprites[:len(out.Pulse)])
	return out, nil
}

// ComposeNativeTownFrame reproduces 0x2cf71's steady redraw. It starts from
// the caller-selected FDOTHER background, redraws the current-label panel and
// FDTXT_000 selection name, applies FDICON pulse sequence 0,1,2,1, then copies
// the native 312x192 viewport to VGA (4,4).
func ComposeNativeTownFrame(
	assets *NativeTownAssets,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	variant, selection, pulse int,
) ([]byte, error) {
	if assets == nil || strings == nil || font == nil ||
		variant < 0 || variant >= nativeTownVariantCount ||
		selection < 0 || selection > 5 || pulse < 0 || pulse > 3 ||
		len(assets.Backgrounds[variant]) != NativeTownWidth*NativeTownHeight {
		return nil, errors.New("campaign: native town state is invalid")
	}
	scene := append([]byte(nil), assets.Backgrounds[variant]...)
	if err := assets.Label.BlitOpaqueAt(
		scene, NativeTownWidth, nativeTownLabelX, nativeTownLabelY, false,
	); err != nil {
		return nil, err
	}
	var err error
	scene, err = ComposeNativeChurchTextAt(
		scene, strings, font, nativeTownTextBase+selection,
		nativeTownTextOffset,
	)
	if err != nil {
		return nil, err
	}
	frame := pulse
	if frame == 3 {
		frame = 1
	}
	if err := assets.Pulse[frame].BlitAt(
		scene, NativeTownWidth,
		nativeTownSelectionX[variant][selection],
		nativeTownSelectionY[variant][selection],
	); err != nil {
		return nil, err
	}
	vga := make([]byte, NativeTownWidth*NativeTownHeight)
	for row := 0; row < nativeTownViewportH; row++ {
		copy(
			vga[(nativeTownViewportY+row)*NativeTownWidth+nativeTownViewportX:],
			scene[row*NativeTownWidth:row*NativeTownWidth+nativeTownViewportW],
		)
	}
	return vga, nil
}
