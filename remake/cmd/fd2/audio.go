// audio.go — BGM 播放(doc 12):只消費 music_catalog.json 驗證過的分離 OGG。
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
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/musiccatalog"
)

var audioCtx *audio.Context

type sfxVoice interface {
	IsPlaying() bool
	Close() error
}

// musicRenderPath 只從通過完整 catalog 驗證的 FM／MT-32 bundle 解析路徑。
// 缺 catalog、錯 hash、少任一 render 或未知 track 時整批失敗即關閉。
func (g *Game) musicRenderPath(track string) (string, error) {
	if g == nil {
		return "", fmt.Errorf("music owner unavailable")
	}
	if g.musicCatalog == nil {
		catalog, err := musiccatalog.Load(assetPath("assets"))
		if err != nil {
			return "", err
		}
		g.musicCatalog = catalog
	}
	return g.musicCatalog.Resolve(g.bgmSource, track)
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
	path, err := g.musicRenderPath(track)
	if err != nil {
		return
	}
	raw, err := os.ReadFile(path)
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

// ── SFX(doc36:FDOTHER#31 的13個非空PCM樣本與一個空尾項)──

// loadSFX 只由分離素材包載入完整FDOTHER#31 OGG bank。
func loadSFX() (map[int][]byte, error) {
	return decodeSeparatedSoundBank(31)
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

// loadSeparatedCommandSoundBanks 載入已閉合的標題與戰鬥指令音效庫。
// 它只消費分離素材包；任一音效庫缺漏或 OGG 解碼失敗時整批拒絕。
func loadSeparatedCommandSoundBanks() (map[int]map[int][]byte, error) {
	resources := []int{50, 53, 77, 80, 82, 83, 84, 85, 86, 87, 88, 90, 91, 92, 93, 94, 95}
	out := make(map[int]map[int][]byte, len(resources))
	for _, resource := range resources {
		decoded, err := decodeSeparatedSoundBank(resource)
		if err != nil {
			return nil, err
		}
		out[resource] = decoded
	}
	return out, nil
}

func decodeSeparatedSoundBank(resource int) (map[int][]byte, error) {
	bank, err := fdother.LoadSeparatedSoundBank(separatedAssetPath("sfx"), resource)
	if err != nil {
		return nil, err
	}
	if audioCtx == nil {
		audioCtx = audio.NewContext(44100)
	}
	decoded := make(map[int][]byte, len(bank.Encoded))
	for subresource, encoded := range bank.Encoded {
		stream, decodeErr := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(encoded))
		if decodeErr != nil {
			return nil, fmt.Errorf("sound bank %d subresource %d: %w", resource, subresource, decodeErr)
		}
		pcm, readErr := io.ReadAll(stream)
		if readErr != nil || len(pcm) == 0 {
			return nil, fmt.Errorf("sound bank %d subresource %d decoded empty: %w", resource, subresource, readErr)
		}
		decoded[subresource] = pcm
	}
	return decoded, nil
}

func (g *Game) installSeparatedCommandSounds(banks map[int]map[int][]byte) {
	g.separatedCommandSFX = banks
	g.sfxTitleMove = banks[77][2]
	g.sfxTitleConfirm = banks[77][1]
	g.sfxCommand24Actor = banks[53][3]
	g.sfxCommand24Target = banks[53][2]
	g.sfxCommandModifier = banks[80][0]
	g.sfxCommand9PlayerPalette = banks[80][0]
	g.sfxCommand9PlayerInitial = banks[80][14]
	g.sfxCommand9PlayerRepeat = banks[80][15]
	g.sfxCommand1012Prelude = banks[80][2]
	g.sfxCommand1012Main = banks[80][13]
	g.sfxCommand0Actor, g.sfxCommand0Target = banks[82][0], banks[82][1]
	g.sfxCommand2Actor, g.sfxCommand2Mode2 = banks[83][0], banks[83][1]
	g.sfxCommand2Mode5, g.sfxCommand2Mode6 = banks[83][2], banks[83][3]
	g.sfxCommand3Actor, g.sfxCommand3Sub1, g.sfxCommand3Sub2 = banks[84][0], banks[84][1], banks[84][2]
	g.sfxCommand4Actor, g.sfxCommand4Target = banks[85][0], banks[85][1]
	g.sfxCommand5Actor, g.sfxCommand5Target = banks[86][0], banks[86][1]
	g.sfxCommand6Actor, g.sfxCommand6Target = banks[87][0], banks[87][1]
	g.sfxCommand6Front, g.sfxCommand6Tail = banks[87][2], banks[87][3]
	g.sfxCommand7Actor, g.sfxCommand7Target = banks[88][0], banks[88][1]
	g.sfxDeath, g.sfxTransition = banks[88][0], banks[88][1]
	g.sfxCommand8Actor, g.sfxCommand8Sub1, g.sfxCommand8Sub2 = banks[90][0], banks[90][1], banks[90][2]
	g.sfxSpawnIntro = banks[95][0]
}

func (g *Game) separatedCommandSound(resource, selector int) []byte {
	if g == nil || g.requireSeparatedCommandSounds(resource, selector) != nil {
		return nil
	}
	return g.separatedCommandSFX[resource][selector]
}

func (g *Game) requireSeparatedCommandSounds(resource int, samples ...int) error {
	if g == nil {
		return fmt.Errorf("separated FDOTHER #%d sound owner unavailable", resource)
	}
	if g.separatedCommandSFX == nil {
		banks, err := loadSeparatedCommandSoundBanks()
		if err != nil {
			return err
		}
		g.installSeparatedCommandSounds(banks)
	}
	bank := g.separatedCommandSFX[resource]
	if bank == nil {
		return fmt.Errorf("separated FDOTHER #%d sound bank unavailable", resource)
	}
	for _, sample := range samples {
		if len(bank[sample]) == 0 {
			return fmt.Errorf("separated FDOTHER #%d sub%d unavailable", resource, sample)
		}
	}
	return nil
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
