// audio.go — BGM 播放(doc 12):OGG(MT-32 預錄,assets/music/FDMUS_NNN.ogg,玩家自備原版轉出)。
// 忠實 play_bgm(0x25977)語意:同曲不重播;換曲=釋放舊曲再依 Miles loop count 播放。
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

type sfxVoice interface {
	IsPlaying() bool
	Close() error
}

// musicPath 依音源設定回傳曲檔路徑:assets/music_<source>/,缺檔則 fallback 到 assets/music/
// (單資料夾佈局的舊行為/玩家只備一套時)。
func musicPath(source, track string) string {
	if source != "" {
		p := assetPath("assets/music_" + source + "/" + track + ".ogg")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return assetPath("assets/music/" + track + ".ogg")
}

// playBGM 播指定曲(如 "FDMUS_008");同曲不重播;檔案缺失/解碼失敗靜默略過。
// FD2_MUTE=1 或截圖模式(headless 無音訊裝置)不播。
func (g *Game) playBGM(track string) {
	g.playBGMCount(track, 0)
}

// playBGMCount reproduces 0x25977(track, loopCount). Miles loop count zero is
// indefinite; one plays the decoded sequence once. The native same-track
// early return ignores a changed loop count, so bgmCur remains the sole gate.
func (g *Game) playBGMCount(track string, loopCount int) {
	if g != nil && !g.currentNativeSystemOptions().MusicEnabled() {
		if track != "" {
			g.nativeSystemBGMTrack = track
		}
		return
	}
	if track == "" || track == g.bgmCur || os.Getenv("FD2_MUTE") != "" || g.shotPath != "" {
		return
	}
	if loopCount < 0 || loopCount > 1 {
		return
	}
	raw, err := os.ReadFile(musicPath(g.bgmSource, track))
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
	var source io.Reader = s
	if loopCount == 0 {
		source = audio.NewInfiniteLoop(s, s.Length())
	}
	p, err := audioCtx.NewPlayer(source)
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

// stopBGM is the proven play_bgm(track=-1) operation used by original
// cutscene handlers.  Clearing bgmCur allows the following explicit BGM beat
// to restart the same track.
func (g *Game) stopBGM() {
	if g.bgm != nil {
		g.bgm.Close()
		g.bgm = nil
	}
	g.bgmCur = ""
	g.nativeSystemBGMTrack = ""
}

func (g *Game) muteNativeSystemBGM() {
	if g.bgmCur != "" {
		g.nativeSystemBGMTrack = g.bgmCur
	}
	if g.bgm != nil {
		g.bgm.Close()
		g.bgm = nil
	}
	g.bgmCur = ""
}

// ── SFX(doc36:FDOTHER#31 的 14 個 PCM 樣本,tools/export_sfx.py 導出 WAV)──

// loadSFX 載入 assets/sfx/sfx_NN.wav 為 PCM bytes(解碼一次,播放時 NewPlayerFromBytes)。
func loadSFX() map[int][]byte {
	out := map[int][]byte{}
	for i := 0; i < 14; i++ {
		raw, err := os.ReadFile(assetPath(fmt.Sprintf("assets/sfx/sfx_%02d.wav", i)))
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
	raw, err := os.ReadFile(assetPath(path))
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
	if b == nil || audioCtx == nil || !g.currentNativeSystemOptions().SFXEnabled() || os.Getenv("FD2_MUTE") != "" || g.shotPath != "" {
		return
	}
	g.playPCMVoice(b)
}

// playSFX 播一個音效(疊播;原版雙 handle 0x26896/0x26945 可同時兩個,這裡不限)。
func (g *Game) playSFX(id int) {
	if g.sfx == nil || !g.currentNativeSystemOptions().SFXEnabled() || os.Getenv("FD2_MUTE") != "" || g.shotPath != "" {
		return
	}
	b, ok := g.sfx[id]
	if !ok || audioCtx == nil {
		return
	}
	g.playPCMVoice(b)
}

func (g *Game) playPCMVoice(b []byte) {
	if g == nil || len(b) == 0 || audioCtx == nil {
		return
	}
	player := audio.NewPlayerFromBytes(audioCtx, b)
	g.sfxVoices = append(g.sfxVoices, player)
	player.Play()
}

// stepSFXVoices 每幀回收已自然結束的短音效；仍在播放者保持獨立 voice，
// 讓原版不同 handle 的疊播不會被後一個 sample 截斷。
func (g *Game) stepSFXVoices() {
	if g == nil || len(g.sfxVoices) == 0 {
		return
	}
	active := g.sfxVoices[:0]
	for _, voice := range g.sfxVoices {
		if voice == nil {
			continue
		}
		if voice.IsPlaying() {
			active = append(active, voice)
			continue
		}
		_ = voice.Close()
	}
	for i := len(active); i < len(g.sfxVoices); i++ {
		g.sfxVoices[i] = nil
	}
	g.sfxVoices = active
}

func (g *Game) closeAudioPlayers() {
	if g == nil {
		return
	}
	if g.bgm != nil {
		_ = g.bgm.Close()
		g.bgm = nil
	}
	for _, voice := range g.sfxVoices {
		if voice != nil {
			_ = voice.Close()
		}
	}
	g.sfxVoices = nil
}
