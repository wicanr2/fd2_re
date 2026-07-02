// audio.go — BGM 播放(doc 12):OGG(MT-32 預錄,assets/music/FDMUS_NNN.ogg,玩家自備原版轉出)。
// 忠實 play_bgm(0x26777)語意:同曲不重播;換曲=釋放舊曲再播新曲(無限迴圈)。
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

var audioCtx *audio.Context

// playBGM 播指定曲(如 "FDMUS_008");同曲不重播;檔案缺失/解碼失敗靜默略過。
// FD2_MUTE=1 或截圖模式(headless 無音訊裝置)不播。
func (g *Game) playBGM(track string) {
	if track == "" || track == g.bgmCur || os.Getenv("FD2_MUTE") != "" || g.shotPath != "" {
		return
	}
	raw, err := os.ReadFile("assets/music/" + track + ".ogg")
	if err != nil {
		return
	}
	if audioCtx == nil {
		audioCtx = audio.NewContext(44100)
	}
	s, err := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(raw))
	if err != nil {
		return
	}
	p, err := audioCtx.NewPlayer(audio.NewInfiniteLoop(s, s.Length()))
	if err != nil {
		return
	}
	if g.bgm != nil {
		g.bgm.Close()
	}
	g.bgm = p
	g.bgmCur = track
	p.Play()
}

// ── SFX(doc36:FDOTHER#31 的 14 個 PCM 樣本,tools/export_sfx.py 導出 WAV)──

// loadSFX 載入 assets/sfx/sfx_NN.wav 為 PCM bytes(解碼一次,播放時 NewPlayerFromBytes)。
func loadSFX() map[int][]byte {
	out := map[int][]byte{}
	for i := 0; i < 14; i++ {
		raw, err := os.ReadFile(fmt.Sprintf("assets/sfx/sfx_%02d.wav", i))
		if err != nil {
			continue
		}
		if audioCtx == nil {
			audioCtx = audio.NewContext(44100)
		}
		s, err := wav.DecodeWithSampleRate(44100, bytes.NewReader(raw))
		if err != nil {
			continue
		}
		b, err := io.ReadAll(s)
		if err != nil {
			continue
		}
		out[i] = b
	}
	return out
}

// loadWav 載單一 WAV 為 PCM bytes(戰鬥池等零散樣本用)。失敗回 nil。
func loadWav(path string) []byte {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if audioCtx == nil {
		audioCtx = audio.NewContext(44100)
	}
	s, err := wav.DecodeWithSampleRate(44100, bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b, err := io.ReadAll(s)
	if err != nil {
		return nil
	}
	return b
}

// playRaw 直接播 PCM bytes(nil 安全)。
func (g *Game) playRaw(b []byte) {
	if b == nil || audioCtx == nil || os.Getenv("FD2_MUTE") != "" || g.shotPath != "" {
		return
	}
	audio.NewPlayerFromBytes(audioCtx, b).Play()
}

// playSFX 播一個音效(疊播;原版雙 handle 0x26896/0x26945 可同時兩個,這裡不限)。
func (g *Game) playSFX(id int) {
	if g.sfx == nil || os.Getenv("FD2_MUTE") != "" || g.shotPath != "" {
		return
	}
	b, ok := g.sfx[id]
	if !ok || audioCtx == nil {
		return
	}
	audio.NewPlayerFromBytes(audioCtx, b).Play()
}
