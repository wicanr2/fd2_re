// fd2-r5-audio-pcm：R5 階段 4（docs/goal/121，代號 r5o）的 Go 音訊黃金資料。只輸出資料，不改任何既有 Go 檔。
//
// 以凍結 Go 版 cmd/fd2 實際的解碼路徑把 BGM 與音效解成 PCM：
//   - OGG：vorbis.DecodeWithSampleRate(44100, bytes.NewReader(raw)) 後 io.ReadAll（audio.go 的 playBGMCount、
//     decodeSeparatedSoundBank；BGM 由 audio.NewInfiniteLoop 循環，第一輪的位元組與 ReadAll 相同）；
//   - WAV：wav.DecodeWithSampleRate(44100, …) 後 io.ReadAll（audio.go 的 loadWav）。
//
// 另外輸出不重取樣的中間層（vorbis／wav.DecodeWithoutResampling，Ebiten 轉 16 位元、單聲道複製成雙聲道之後、
// 重取樣之前），讓 Rust 端分辨差異來自解碼器還是重取樣。
//
// 輸入：-assets（remake/assets：music_catalog.json 的 FM／MT-32 render 與 sfx/*.wav）、-sfx（分離素材包的 sfx/，
// 每個 FDOTHER_NNN/sample_*.ogg）。輸出：-out 目錄下的 go-pcm.jsonl（每檔一列）與 pcm/<id>.{final,native}.s16
// （16 位元有號、小端、雙聲道交錯）。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100

type stage struct {
	Bytes  int    `json:"bytes"`
	Frames int    `json:"frames"`
	SHA256 string `json:"sha256"`
	Error  string `json:"error,omitempty"`
}

type record struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Path         string `json:"path"`
	SourceBytes  int    `json:"source_bytes"`
	SourceSHA256 string `json:"source_sha256"`
	Final        stage  `json:"final"`
	Native       stage  `json:"native"`
}

type input struct{ id, kind, path, rel string }

func main() {
	assets := flag.String("assets", "", "remake/assets")
	sfx := flag.String("sfx", "", "分離素材包的 sfx/")
	out := flag.String("out", "", "輸出目錄")
	flag.Parse()
	if *assets == "" || *sfx == "" || *out == "" {
		log.Fatal("需要 -assets、-sfx、-out")
	}
	inputs, err := listInputs(*assets, *sfx)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(*out, "pcm"), 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(filepath.Join(*out, "go-pcm.jsonl"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, in := range inputs {
		raw, err := os.ReadFile(in.path)
		if err != nil {
			log.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		rec := record{ID: in.id, Kind: in.kind, Path: in.rel, SourceBytes: len(raw), SourceSHA256: hex.EncodeToString(sum[:])}
		base := filepath.Join(*out, "pcm", strings.ReplaceAll(in.id, "/", "__"))
		rec.Final = decodeStage(raw, in.kind, true, base+".final.s16")
		rec.Native = decodeStage(raw, in.kind, false, base+".native.s16")
		if err := enc.Encode(rec); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("fd2-r5-audio-pcm: %d 檔\n", len(inputs))
}

// decodeStage 照 audio.go 解碼；resample=false 時改用 DecodeWithoutResampling。
func decodeStage(raw []byte, kind string, resample bool, path string) stage {
	var r io.Reader
	var err error
	switch {
	case kind == "wav" && resample:
		r, err = wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(raw))
	case kind == "wav":
		r, err = wav.DecodeWithoutResampling(bytes.NewReader(raw))
	case resample:
		r, err = vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(raw))
	default:
		r, err = vorbis.DecodeWithoutResampling(bytes.NewReader(raw))
	}
	if err != nil {
		return stage{Error: err.Error()}
	}
	pcm, err := io.ReadAll(r)
	if err != nil {
		return stage{Error: err.Error()}
	}
	if err := os.WriteFile(path, pcm, 0o644); err != nil {
		log.Fatal(err)
	}
	sum := sha256.Sum256(pcm)
	return stage{Bytes: len(pcm), Frames: len(pcm) / 4, SHA256: hex.EncodeToString(sum[:])}
}

// listInputs：BGM 依 music_catalog.json 的曲目與 render（fm、mt32），音效 OGG 依 sfx/FDOTHER_NNN/sample_*.ogg，
// WAV 依 assets/sfx/*.wav；各類以 id 排序。
func listInputs(assets, sfx string) ([]input, error) {
	raw, err := os.ReadFile(filepath.Join(assets, "music_catalog.json"))
	if err != nil {
		return nil, err
	}
	var catalog struct {
		Tracks []struct {
			TrackID string `json:"track_id"`
			Renders map[string]struct {
				Path string `json:"path"`
			} `json:"renders"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, err
	}
	var bgm, sfxOGG, sfxWAV []input
	for _, t := range catalog.Tracks {
		for profile, render := range t.Renders {
			bgm = append(bgm, input{id: "bgm/" + profile + "/" + t.TrackID, kind: "bgm", path: filepath.Join(assets, render.Path), rel: "assets/" + render.Path})
		}
	}
	oggs, err := filepath.Glob(filepath.Join(sfx, "FDOTHER_*", "sample_*.ogg"))
	if err != nil {
		return nil, err
	}
	for _, p := range oggs {
		bank, name := filepath.Base(filepath.Dir(p)), strings.TrimSuffix(filepath.Base(p), ".ogg")
		sfxOGG = append(sfxOGG, input{id: "sfx/" + bank + "/" + name, kind: "sfx", path: p, rel: "sfx/" + bank + "/" + filepath.Base(p)})
	}
	wavs, err := filepath.Glob(filepath.Join(assets, "sfx", "*.wav"))
	if err != nil {
		return nil, err
	}
	for _, p := range wavs {
		sfxWAV = append(sfxWAV, input{id: "wav/" + strings.TrimSuffix(filepath.Base(p), ".wav"), kind: "wav", path: p, rel: "assets/sfx/" + filepath.Base(p)})
	}
	var all []input
	for _, group := range [][]input{bgm, sfxOGG, sfxWAV} {
		sort.Slice(group, func(i, j int) bool { return group[i].id < group[j].id })
		all = append(all, group...)
	}
	if len(bgm) == 0 || len(sfxOGG) == 0 || len(sfxWAV) == 0 {
		return nil, fmt.Errorf("輸入不完整：bgm %d、sfx %d、wav %d", len(bgm), len(sfxOGG), len(sfxWAV))
	}
	return all, nil
}
