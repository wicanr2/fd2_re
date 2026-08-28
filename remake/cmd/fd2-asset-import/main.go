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
}

func verifyFDOTHER(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	md5sum, sha := md5.Sum(raw), sha256.Sum256(raw)
	if len(raw) != fdotherSize || hex.EncodeToString(md5sum[:]) != fdotherMD5 || hex.EncodeToString(sha[:]) != fdotherSHA256 {
		return fmt.Errorf("FDOTHER.DAT 版本不符：size=%d md5=%x sha256=%x", len(raw), md5sum, sha)
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
			MD5: fdotherMD5, SHA256: fdotherSHA256,
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
	outputRoot := flag.String("out", "", "分離素材包根目錄")
	flag.Parse()
	if *fdotherPath == "" || *outputRoot == "" {
		fmt.Fprintln(os.Stderr, "用法：fd2-asset-import -fdother FDOTHER.DAT -out ASSET_PACK")
		os.Exit(2)
	}
	if err := exportCommandGrid(*fdotherPath, *outputRoot); err != nil {
		fmt.Fprintln(os.Stderr, "匯入失敗：", err)
		os.Exit(1)
	}
	fmt.Println("已匯出戰場指令格：256色調色盤＋78張 PNG")
}
