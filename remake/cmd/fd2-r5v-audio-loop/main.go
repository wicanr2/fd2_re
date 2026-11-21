// fd2-r5v-audio-loop：R5 階段 4（docs/goal/121，代號 r5v）的 Go 音訊黃金資料。只輸出資料，不改任何既有 Go 檔。
//
// 兩種輸出：
//   - 預設：music_catalog.json 的每個 BGM render 照 cmd/fd2 audio.go 的 playBGMCount（loopCount == 0）以
//     vorbis.DecodeWithSampleRate(44100, …) 後 audio.NewInfiniteLoop(s, s.Length()) 讀兩圈，輸出 Length() 與
//     兩圈各自的 SHA-256（loop.jsonl）。Rust 外殼以 Bevy 的 LOOP 重播第一圈，用這份資料核對 Go 的第二圈與第一圈相同。
//   - -lookup1：jfreymuth/vorbis v1.0.2 codebook.go 的 lookup1Values 公式
//     int(math.Floor(math.Pow(float64(entries), 1/float64(dim))))，在本機 Go（amd64 的 math.Pow 經組語 Exp／Log）
//     下的值。表一：dim 1..16 × entries 1..65536，輸出以 int64 小端串接後的 SHA-256；表二：dim 2..24 的每個
//     整數次方 r^dim ≤ 2^24 及其 ±1，逐筆輸出（lookup1.json）。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

type loopRecord struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Length      int64  `json:"length"`
	Pass1SHA256 string `json:"pass1_sha256"`
	Pass2SHA256 string `json:"pass2_sha256"`
}

type lookupCase struct {
	Entries int `json:"entries"`
	Dim     int `json:"dim"`
	Value   int `json:"value"`
}

func lookup1Values(entries int, dim uint32) int {
	return int(math.Floor(math.Pow(float64(entries), 1/float64(dim))))
}

func main() {
	assets := flag.String("assets", "", "remake/assets")
	out := flag.String("out", "", "輸出目錄")
	lookup1 := flag.Bool("lookup1", false, "改為輸出 lookup1Values 對照表")
	flag.Parse()
	if *out == "" || (!*lookup1 && *assets == "") {
		log.Fatal("需要 -out，以及 -assets 或 -lookup1")
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	if *lookup1 {
		writeLookup1(filepath.Join(*out, "lookup1.json"))
		return
	}
	writeLoops(*assets, filepath.Join(*out, "loop.jsonl"))
}

func writeLookup1(path string) {
	h := sha256.New()
	var b [8]byte
	for dim := uint32(1); dim <= 16; dim++ {
		for entries := 1; entries <= 65536; entries++ {
			binary.LittleEndian.PutUint64(b[:], uint64(int64(lookup1Values(entries, dim))))
			h.Write(b[:])
		}
	}
	var cases []lookupCase
	for dim := 2; dim <= 24; dim++ {
		for r := 1; ; r++ {
			p := 1
			for i := 0; i < dim; i++ {
				p *= r
			}
			if p > 1<<24 {
				break
			}
			for _, e := range []int{p - 1, p, p + 1} {
				if e >= 1 {
					cases = append(cases, lookupCase{Entries: e, Dim: dim, Value: lookup1Values(e, uint32(dim))})
				}
			}
		}
	}
	doc := map[string]any{
		"schema":        "fd2-r5v-lookup1/1",
		"table_dims":    16,
		"table_entries": 65536,
		"table_sha256":  hex.EncodeToString(h.Sum(nil)),
		"powers":        cases,
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("fd2-r5v-audio-loop -lookup1: %d 筆整數次方\n", len(cases))
}

func writeLoops(assets, path string) {
	raw, err := os.ReadFile(filepath.Join(assets, "music_catalog.json"))
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	type input struct{ id, path, rel string }
	var inputs []input
	for _, t := range catalog.Tracks {
		for profile, render := range t.Renders {
			inputs = append(inputs, input{"bgm/" + profile + "/" + t.TrackID, filepath.Join(assets, render.Path), "assets/" + render.Path})
		}
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].id < inputs[j].id })
	if len(inputs) == 0 {
		log.Fatal("沒有 BGM")
	}
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, in := range inputs {
		src, err := os.ReadFile(in.path)
		if err != nil {
			log.Fatal(err)
		}
		s, err := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(src))
		if err != nil {
			log.Fatalf("%s: %v", in.id, err)
		}
		loop := audio.NewInfiniteLoop(s, s.Length())
		buf := make([]byte, 2*s.Length())
		if _, err := io.ReadFull(loop, buf); err != nil {
			log.Fatalf("%s: %v", in.id, err)
		}
		p1 := sha256.Sum256(buf[:s.Length()])
		p2 := sha256.Sum256(buf[s.Length():])
		rec := loopRecord{ID: in.id, Path: in.rel, Length: s.Length(), Pass1SHA256: hex.EncodeToString(p1[:]), Pass2SHA256: hex.EncodeToString(p2[:])}
		if err := enc.Encode(rec); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("fd2-r5v-audio-loop: %d 個 BGM\n", len(inputs))
}
