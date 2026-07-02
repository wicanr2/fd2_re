// audio.go — BGM 播放(doc 12):OGG(MT-32 預錄,assets/music/FDMUS_NNN.ogg,玩家自備原版轉出)。
// 忠實 play_bgm(0x26777)語意:同曲不重播;換曲=釋放舊曲再播新曲(無限迴圈)。
package main

import (
	"bytes"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
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
