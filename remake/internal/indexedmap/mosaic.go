package indexedmap

import "errors"

// PhaseBannerSteps 是原版回合橫幅的步數。
//
// dosgolem 逐幀收據（docs/data/ui-traces/fd2-phase-banner-20260909.json）：
// 33 張畫面的馬賽克方塊邊長是 `1 + min(f, 32-f)`，由 1 漲到 17 再縮回 1；
// 第 f 張與第 32-f 張逐位元組相同，第 32 張與第 0 張相同。整段期間地圖完全
// 不重繪，所以馬賽克是對**保留下來的已合成畫面**每一步重算，不是逐步累積。
// `PLAYER PHASE` 與 `ENEMY PHASE` 共用同一段排程。
const PhaseBannerSteps = 33

// PhaseBannerBlock 回傳第 step 步的方塊邊長。
func PhaseBannerBlock(step int) int {
	if step < 0 || step >= PhaseBannerSteps {
		return 1
	}
	other := PhaseBannerSteps - 1 - step
	if other < step {
		return 1 + other
	}
	return 1 + step
}

// MosaicNativeMapViewport 把 src 的 312×192 地圖窗格以 block 為邊長馬賽克化後
// 寫進 dst，窗格外（四邊各 4 px 的黑邊）原樣保留。
//
// 取樣規則是**方塊左上角**那一個來源像素：對邊長 4／8／16 逐方塊比對原始幀，
// 1872/1872、468/468、114/114 全中，中心與右下角則只有個位數百分比。格線原點
// 就是窗格原點 (4,4)。block 為 1 時等同整幀複製。
func MosaicNativeMapViewport(dst, src []byte, block int) error {
	if len(dst) < NativeMapVGASize || len(src) < NativeMapVGASize {
		return errors.New("indexedmap: mosaic buffers are smaller than one native frame")
	}
	if block < 1 || block > steadyViewportWidth {
		return errors.New("indexedmap: mosaic block is outside the viewport")
	}
	copy(dst[:NativeMapVGASize], src[:NativeMapVGASize])
	if block == 1 {
		return nil
	}
	const originX, originY = 4, 4
	for y := 0; y < viewHeight; y += block {
		for x := 0; x < steadyViewportWidth; x += block {
			v := src[(originY+y)*viewWidth+originX+x]
			for dy := 0; dy < block && y+dy < viewHeight; dy++ {
				row := (originY + y + dy) * viewWidth
				for dx := 0; dx < block && x+dx < steadyViewportWidth; dx++ {
					dst[row+originX+x+dx] = v
				}
			}
		}
	}
	return nil
}
