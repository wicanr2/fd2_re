package main

import (
	"errors"
	"image"
	"log"
	"sync"
)

// 主題包載入時延後到用到才做的檢查（#317）。載入仍逐檔核對存在、路徑、SHA-256 與 PNG 標頭
// 尺寸，缺件或雜湊不符照舊整包拒收；只有必須解碼像素才能判斷的透明度條件延到第一次預備。
// 延後的檢查若失敗，該筆資源退回原版畫格並由 Update 顯示訊息，不 panic、不發布半組圖。

// errModernHDDeferredRejected 標示延後到第一次使用的檢查沒有通過。符合這個錯誤時，呼叫端把那筆
// 資源當成包內沒有登錄，改用原版畫格；其他錯誤（檔案在載入後不見或被改）照舊讓交易失敗。
var errModernHDDeferredRejected = errors.New("HD 素材延後檢查未通過")

// modernHDAlphaCoverage 摘要一張圖的透明度：有沒有完全透明、完全不透明與任何非零 alpha 的像素。
type modernHDAlphaCoverage struct {
	transparent bool
	opaque      bool
	visible     bool
}

// modernHDAlphaCoverageOf 與逐點 At().RGBA() 的判斷相同；常見的 NRGBA／RGBA 直接讀 Pix，
// 找到完全透明與完全不透明兩端後就停（此時 visible 必為真）。
func modernHDAlphaCoverageOf(picture image.Image) modernHDAlphaCoverage {
	var c modernHDAlphaCoverage
	var pix []byte
	stride, rect := 0, picture.Bounds()
	switch p := picture.(type) {
	case *image.NRGBA:
		pix, stride = p.Pix, p.Stride
	case *image.RGBA:
		pix, stride = p.Pix, p.Stride
	}
	if pix != nil {
		width := rect.Dx() * 4
		for y := 0; y < rect.Dy(); y++ {
			row := pix[y*stride : y*stride+width]
			for x := 3; x < len(row); x += 4 {
				a := row[x]
				c.transparent = c.transparent || a == 0
				c.opaque = c.opaque || a == 0xff
				c.visible = c.visible || a != 0
				if c.transparent && c.opaque {
					return c
				}
			}
		}
		return c
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			_, _, _, a := picture.At(x, y).RGBA()
			c.transparent = c.transparent || a == 0
			c.opaque = c.opaque || a == 0xffff
			c.visible = c.visible || a != 0
			if c.transparent && c.opaque {
				return c
			}
		}
	}
	return c
}

// modernThemeNotices 收集延後檢查的拒收訊息，等 Update 取走後顯示給玩家。
type modernThemeNotices struct {
	mu      sync.Mutex
	pending []string
}

func (set *modernStoryPortraitSet) themeNotices() *modernThemeNotices {
	if set == nil {
		return nil
	}
	if set.notices == nil {
		set.notices = &modernThemeNotices{}
	}
	return set.notices
}

func (n *modernThemeNotices) reject(err error) {
	log.Printf("HD 素材延後檢查拒收，改用原版畫格：%v", err)
	if n == nil {
		return
	}
	n.mu.Lock()
	n.pending = append(n.pending, err.Error())
	n.mu.Unlock()
}

func (n *modernThemeNotices) take() []string {
	if n == nil {
		return nil
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	out := n.pending
	n.pending = nil
	return out
}

// modernThemeDeferredMessage 是延後檢查拒收時給玩家看的訊息。
const modernThemeDeferredMessage = "HD 素材有一組未通過檢查，已改用原版圖："

// surfaceModernThemeNotices 在 Update 開頭把延後檢查的拒收訊息顯示出來。
func (g *Game) surfaceModernThemeNotices() {
	seen := map[*modernThemeNotices]bool{}
	for _, set := range []*modernStoryPortraitSet{g.modernStoryPortraits, g.modernThemeCache} {
		if set == nil || set.notices == nil || seen[set.notices] {
			continue
		}
		seen[set.notices] = true
		if pending := set.notices.take(); len(pending) != 0 {
			g.msg = modernThemeDeferredMessage + pending[len(pending)-1]
		}
	}
}
