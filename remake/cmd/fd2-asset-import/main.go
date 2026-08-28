// fd2-asset-import 將玩家合法持有的原版資源轉成正式 runtime 可消費的分離素材。
// 原版 archive reader 只存在於這個離線命令，不應被遊戲 renderer 呼叫。
package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	fdotherSize   = 3382481
	fdotherMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	fdotherSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	actionCells   = 78
)

type paletteDocument struct {
	SchemaVersion int      `json:"schema_version"`
	AssetID       string   `json:"asset_id"`
	Source        sourceID `json:"source"`
	Components    []int    `json:"dac_6bit_components"`
}

type sourceID struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type surfaceDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	AssetID       string   `json:"asset_id"`
	Status        string   `json:"status"`
	Codec         string   `json:"codec,omitempty"`
	Width         int      `json:"width,omitempty"`
	Height        int      `json:"height,omitempty"`
	Frame         string   `json:"frame,omitempty"`
	Mask          string   `json:"mask,omitempty"`
	Evidence      string   `json:"evidence"`
	ReasonCode    string   `json:"reason_code,omitempty"`
	Source        sourceID `json:"source"`
}

type archiveIdentity struct {
	file, prefix string
	size         int
	md5, sha256  string
}

var surfaceArchives = []archiveIdentity{
	{file: "BG.DAT", prefix: "BG", size: 624564, md5: "4b5414c92b40ef25ba0ee10c80f9e149", sha256: "b9fc21d019d6256a4bb7e6da1cefcb0bfe331d8ff74a52a8201570afc98b56de"},
	{file: "TAI.DAT", prefix: "TAI", size: 94917, md5: "7cfe4b9ad2cbff44b2ebd7ab2f94e4aa", sha256: "d56fea9c43f8bb59aad89ad76698885d7e07f380d12a4547888a0b60ea5e0410"},
}

func verifyFDOTHER(path string) error {
	return verifyArchive(path, archiveIdentity{file: "FDOTHER.DAT", size: fdotherSize, md5: fdotherMD5, sha256: fdotherSHA256})
}

func verifyArchive(path string, identity archiveIdentity) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	md5sum, sha := md5.Sum(raw), sha256.Sum256(raw)
	if len(raw) != identity.size || hex.EncodeToString(md5sum[:]) != identity.md5 || hex.EncodeToString(sha[:]) != identity.sha256 {
		return fmt.Errorf("%s 版本不符：size=%d md5=%x sha256=%x", identity.file, len(raw), md5sum, sha)
	}
	return nil
}

func exportSingleFrameArchive(path, outputRoot string, identity archiveIdentity) (int, int, error) {
	if err := verifyArchive(path, identity); err != nil {
		return 0, 0, err
	}
	count, err := fdother.ArchiveResourceCount(path)
	if err != nil {
		return 0, 0, err
	}
	decoded, blocked := 0, 0
	for resource := 0; resource < count; resource++ {
		raw, readErr := fdother.ReadResource(path, resource)
		directory := filepath.Join(outputRoot, "surfaces", fmt.Sprintf("%s_%03d", identity.prefix, resource))
		doc := surfaceDocument{
			SchemaVersion: 1, Kind: "indexed_surface", AssetID: fmt.Sprintf("surface/%s_%03d", identity.prefix, resource),
			Status: "blocked", Evidence: "confirmed", ReasonCode: "decode_failed",
			Source: sourceID{File: identity.file, Resource: resource, Size: identity.size, MD5: identity.md5, SHA256: identity.sha256, RawSize: len(raw)},
		}
		if readErr == nil {
			doc.Source.Size = identity.size
			frame, decodeErr := fdother.ParseSingleFrame(raw)
			if decodeErr == nil {
				indexed, mask, layerErr := frame.IndexedLayers()
				if layerErr == nil {
					doc.Status, doc.Codec, doc.Width, doc.Height = "decoded", "fd2_4e63d_single_frame", frame.Width, frame.Height
					doc.Frame, doc.Mask, doc.ReasonCode = "frame.png", "mask.png", ""
					if err := writeSurfacePNGs(directory, frame.Width, frame.Height, indexed, mask); err != nil {
						return decoded, blocked, err
					}
					decoded++
				} else {
					blocked++
				}
			} else {
				blocked++
			}
		} else {
			blocked++
		}
		if err := writeJSON(filepath.Join(directory, "resource.json"), doc); err != nil {
			return decoded, blocked, err
		}
	}
	return decoded, blocked, nil
}

func writeSurfacePNGs(directory string, width, height int, indexed, mask []byte) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 0xff}
	}
	frame := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	copy(frame.Pix, indexed)
	maskImage := image.NewGray(image.Rect(0, 0, width, height))
	copy(maskImage.Pix, mask)
	for name, value := range map[string]image.Image{"frame.png": frame, "mask.png": maskImage} {
		file, err := os.Create(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		encodeErr := png.Encode(file, value)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func exportCommandGrid(fdotherPath, outputRoot string) error {
	if err := verifyFDOTHER(fdotherPath); err != nil {
		return err
	}
	dac, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		return err
	}
	if _, err := fdother.ParseVGAPalette(dac); err != nil {
		return err
	}
	components := make([]int, len(dac))
	for i, component := range dac {
		components[i] = int(component)
	}
	document := paletteDocument{
		SchemaVersion: 1,
		AssetID:       "palette/fdother_000",
		Source: sourceID{
			File: "FDOTHER.DAT", Resource: 0, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: len(dac),
		},
		Components: components,
	}
	if err := writeJSON(filepath.Join(outputRoot, "palette", "fdother_000.json"), document); err != nil {
		return err
	}

	palette, _ := fdother.ParseVGAPalette(dac)
	cells, err := fdother.DecodeRawCellResource(fdotherPath, 2)
	if err != nil {
		return err
	}
	if len(cells) != actionCells {
		return fmt.Errorf("FDOTHER #2 cells=%d, want %d", len(cells), actionCells)
	}
	directory := filepath.Join(outputRoot, "ui", "action_cells")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	for index, cell := range cells {
		image, err := cell.Paletted(palette)
		if err != nil {
			return fmt.Errorf("cell %d: %w", index, err)
		}
		path := filepath.Join(directory, fmt.Sprintf("cell_%03d.png", index))
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		encodeErr := png.Encode(file, image)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func main() {
	fdotherPath := flag.String("fdother", "", "固定版本 FDOTHER.DAT 路徑")
	bgPath := flag.String("bg", "", "固定版本 BG.DAT 路徑")
	taiPath := flag.String("tai", "", "固定版本 TAI.DAT 路徑")
	outputRoot := flag.String("out", "", "分離素材包根目錄")
	flag.Parse()
	if *outputRoot == "" || (*fdotherPath == "" && *bgPath == "" && *taiPath == "") {
		fmt.Fprintln(os.Stderr, "用法：fd2-asset-import [-fdother FDOTHER.DAT] [-bg BG.DAT] [-tai TAI.DAT] -out ASSET_PACK")
		os.Exit(2)
	}
	if *fdotherPath != "" {
		if err := exportCommandGrid(*fdotherPath, *outputRoot); err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Println("已匯出戰場指令格：256色調色盤＋78張 PNG")
	}
	for _, request := range []struct {
		path     string
		identity archiveIdentity
	}{{*bgPath, surfaceArchives[0]}, {*taiPath, surfaceArchives[1]}} {
		if request.path == "" {
			continue
		}
		decoded, blocked, err := exportSingleFrameArchive(request.path, *outputRoot, request.identity)
		if err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Printf("已匯出 %s：decoded=%d blocked=%d\n", request.identity.file, decoded, blocked)
	}
}
