package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	separatedFDOTHERSize   = 3382481
	separatedFDOTHERMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	separatedFDOTHERSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
)

type separatedShopEntry struct {
	Index  int    `json:"index"`
	Role   string `json:"role"`
	Codec  string `json:"codec"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Frame  string `json:"frame"`
	Mask   string `json:"mask"`
}

type separatedShopSuccess struct {
	X               int                  `json:"x"`
	Y               int                  `json:"y"`
	PreTicks        int                  `json:"pre_ticks"`
	TicksPerFrame   int                  `json:"ticks_per_frame"`
	PostTicks       int                  `json:"post_ticks"`
	RestorePortrait bool                 `json:"restore_portrait"`
	Frames          []separatedShopEntry `json:"frames"`
}

type separatedShopDocument struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	AssetID       string `json:"asset_id"`
	Status        string `json:"status"`
	Evidence      string `json:"evidence"`
	ContainerKind string `json:"container_kind"`
	EntryCount    int    `json:"entry_count"`
	Source        struct {
		File     string `json:"file"`
		Resource int    `json:"resource"`
		Size     int    `json:"size"`
		MD5      string `json:"md5"`
		SHA256   string `json:"sha256"`
		RawSize  int    `json:"raw_size"`
	} `json:"source"`
	Entries []separatedShopEntry `json:"entries"`
	Success separatedShopSuccess `json:"success"`
}

func readSeparatedIndexed(path string, width, height int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	p, ok := decoded.(*image.Paletted)
	if !ok || p.Rect != image.Rect(0, 0, width, height) || p.Stride != width || len(p.Pix) != width*height {
		return nil, errors.New("campaign: shop frame is not packed indexed PNG")
	}
	return append([]byte(nil), p.Pix...), nil
}

func readSeparatedBinaryMask(path string, width, height int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	g, ok := decoded.(*image.Gray)
	if !ok || g.Rect != image.Rect(0, 0, width, height) || g.Stride != width || len(g.Pix) != width*height {
		return nil, errors.New("campaign: shop mask is not packed grayscale PNG")
	}
	out := append([]byte(nil), g.Pix...)
	for _, v := range out {
		if v != 0 && v != 255 {
			return nil, errors.New("campaign: shop mask is not binary")
		}
	}
	return out, nil
}

// LoadSeparatedNativeShopAssets loads one caller-proven shop resource without
// consulting FDOTHER.DAT. Production callers must preflight all three variants.
func LoadSeparatedNativeShopAssets(packRoot string, resource int) (*NativeShopAssets, error) {
	wantContainer, ok := map[int]string{12: "llllll", 29: "scene_lmi1", 63: "llllll"}[resource]
	if !ok || packRoot == "" {
		return nil, errors.New("campaign: unsupported separated shop resource")
	}
	dir := filepath.Join(packRoot, "shop", fmt.Sprintf("FDOTHER_%03d", resource))
	raw, err := os.ReadFile(filepath.Join(dir, "resource.json"))
	if err != nil {
		return nil, fmt.Errorf("campaign: separated shop metadata: %w", err)
	}
	var doc separatedShopDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	wantRawCount := map[int]int{12: 28, 29: 24, 63: 30}[resource]
	wantTypedCount := map[int]int{12: 23, 29: 19, 63: 25}[resource]
	if doc.SchemaVersion != 1 || doc.Kind != "native_shop_indexed_assets" || doc.AssetID != fmt.Sprintf("shop/FDOTHER_%03d", resource) || doc.Status != "decoded" || doc.Evidence != "confirmed" || doc.ContainerKind != wantContainer || doc.EntryCount != wantRawCount || len(doc.Entries) != wantTypedCount || doc.Source.File != "FDOTHER.DAT" || doc.Source.Resource != resource || doc.Source.Size != separatedFDOTHERSize || doc.Source.MD5 != separatedFDOTHERMD5 || doc.Source.SHA256 != separatedFDOTHERSHA256 || doc.Source.RawSize <= 0 {
		return nil, errors.New("campaign: separated shop metadata contract mismatch")
	}
	entries := make(map[int]separatedShopEntry, len(doc.Entries))
	for _, e := range doc.Entries {
		if _, dup := entries[e.Index]; dup {
			return nil, errors.New("campaign: duplicate separated shop entry")
		}
		entries[e.Index] = e
	}
	load := func(index int, role, codec string, mask bool) ([]byte, separatedShopEntry, error) {
		e, ok := entries[index]
		wantFrame := filepath.ToSlash(filepath.Join(fmt.Sprintf("entry_%03d", index), "frame.png"))
		if !ok || e.Role != role || e.Codec != codec || e.Width <= 0 || e.Height <= 0 || e.Frame != wantFrame {
			return nil, e, errors.New("campaign: separated shop entry contract mismatch")
		}
		pixels, err := readSeparatedIndexed(filepath.Join(dir, e.Frame), e.Width, e.Height)
		if err != nil {
			return nil, e, err
		}
		if mask {
			if e.Mask != filepath.ToSlash(filepath.Join(fmt.Sprintf("entry_%03d", index), "mask.png")) {
				return nil, e, errors.New("campaign: separated shop mask path mismatch")
			}
			m, err := readSeparatedBinaryMask(filepath.Join(dir, e.Mask), e.Width, e.Height)
			if err != nil {
				return nil, e, err
			}
			for i, v := range pixels {
				want := byte(0)
				if v != 0 {
					want = 255
				}
				if m[i] != want {
					return nil, e, errors.New("campaign: separated shop transparency mask mismatch")
				}
			}
		} else if e.Mask != "" {
			return nil, e, errors.New("campaign: opaque shop entry unexpectedly has mask")
		}
		return pixels, e, nil
	}
	bg, bge, err := load(0, "background", "fd2_4e63d_single_frame", false)
	if err != nil || bge.Width != NativeShopWidth || bge.Height != NativeShopHeight {
		return nil, errors.New("campaign: separated shop background mismatch")
	}
	dec, de, err := load(1, "decoration", "opaque_high_run", false)
	if err != nil || de.Width != 63 || de.Height != 15 {
		return nil, errors.New("campaign: separated shop decoration mismatch")
	}
	gold, ge, err := load(2, "gold_roll_strip", "raw_indexed_transparent", true)
	if err != nil || ge.Width != 6 || ge.Height != 99 {
		return nil, errors.New("campaign: separated shop gold strip mismatch")
	}
	out := &NativeShopAssets{ResourceID: resource, Background: bg, Decoration: fdother.LMI1Entry{Width: de.Width, Height: de.Height, Pixels: dec}, GoldRollStrip: fdother.RawCell{Width: ge.Width, Height: ge.Height, Pixels: gold}}
	for option := 0; option < 4; option++ {
		for variant := 0; variant < 2; variant++ {
			index := 3 + option*2 + variant
			p, e, err := load(index, "service_cell", "raw_indexed_transparent", true)
			if err != nil {
				return nil, err
			}
			out.ServiceCells[option][variant] = fdother.RawCell{Width: e.Width, Height: e.Height, Pixels: p}
		}
	}
	p, pe, err := load(15, "price_cell", "raw_indexed_transparent", true)
	if err != nil {
		return nil, err
	}
	out.PriceCell = fdother.RawCell{Width: pe.Width, Height: pe.Height, Pixels: p}
	p, pe, err = load(16, "panel", "opaque_high_run", false)
	if err != nil {
		return nil, err
	}
	out.Panel = fdother.LMI1Entry{Width: pe.Width, Height: pe.Height, Pixels: p}
	for i := 0; i < 5; i++ {
		p, e, err := load(18+i, "compare_cell", "opaque_high_run", false)
		if err != nil {
			return nil, err
		}
		out.CompareCells[i] = fdother.LMI1Entry{Width: e.Width, Height: e.Height, Pixels: p}
	}
	wantPlan := map[int]separatedShopSuccess{12: {X: 169, Y: 45, TicksPerFrame: 2, RestorePortrait: true}, 29: {X: 148, Y: 39, PreTicks: 1, PostTicks: 8, RestorePortrait: true}, 63: {X: 131, Y: 28, TicksPerFrame: 2}}[resource]
	wantFrames := map[int]int{12: 5, 29: 1, 63: 7}[resource]
	if doc.Success.X != wantPlan.X || doc.Success.Y != wantPlan.Y || doc.Success.PreTicks != wantPlan.PreTicks || doc.Success.TicksPerFrame != wantPlan.TicksPerFrame || doc.Success.PostTicks != wantPlan.PostTicks || doc.Success.RestorePortrait != wantPlan.RestorePortrait || len(doc.Success.Frames) != wantFrames {
		return nil, errors.New("campaign: separated shop success plan mismatch")
	}
	for i, se := range doc.Success.Frames {
		if se.Index != 23+i {
			return nil, errors.New("campaign: separated shop success index mismatch")
		}
		e, ok := entries[se.Index]
		wantFrame := filepath.ToSlash(filepath.Join(fmt.Sprintf("success_%03d", i), "frame.png"))
		wantMask := filepath.ToSlash(filepath.Join(fmt.Sprintf("success_%03d", i), "mask.png"))
		if !ok || e != se || e.Role != "success_frame" || e.Codec != "fd2_4e63d_single_frame" || e.Width <= 0 || e.Height <= 0 || e.Frame != wantFrame || e.Mask != wantMask {
			return nil, errors.New("campaign: separated shop success entry mismatch")
		}
		p, err := readSeparatedIndexed(filepath.Join(dir, e.Frame), e.Width, e.Height)
		if err != nil {
			return nil, err
		}
		m, err := readSeparatedBinaryMask(filepath.Join(dir, e.Mask), e.Width, e.Height)
		if err != nil {
			return nil, err
		}
		out.SuccessFrames = append(out.SuccessFrames, fdother.Frame{Width: e.Width, Height: e.Height, Indexed: p, Mask: m})
	}
	return out, nil
}
