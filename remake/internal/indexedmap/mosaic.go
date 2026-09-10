package indexedmap

import "errors"

// PhaseBannerSteps 是原版回合橫幅的步數。
//
// dosgolem 逐幀收據（docs/data/ui-traces/fd2-phase-banner-20260909.json）：
// 33 張畫面的馬賽克方塊邊長是 `1 + min(f, 32-f)`，由 1 漲到 17 再縮回 1；
// 第 f 張與第 32-f 張逐位元組相同，第 32 張與第 0 張相同。整段期間地圖完全
// 不重繪，所以馬賽克是對**保留下來的已合成畫面**每一步重算，不是逐步累積。
// `PLAYER PHASE` 與 `ENEMY PHASE` 共用同一段排程。
//
// 33 步是兩支函式接起來的：`sub_1F1CC` 進場 16 步（邊長 1..16），`sub_1F30A`
// 退場 17 步（邊長 17..1），中間隔一次 `delay()`。原始指令見
// docs/data/fd2_phase_banner_presentation_disasm.txt，量測見
// docs/knowledge-base/105-phase-banner-timing-20260910.md。
const PhaseBannerSteps = 33

// PhaseBannerEnterSteps 是 `sub_1F1CC` 進場迴圈的步數（`ebp` ＝ 1..16）。
// 之後才是中段 `delay()`，再進 `sub_1F30A` 的 PhaseBannerExitSteps 步。
const PhaseBannerEnterSteps = 16

// PhaseBannerExitSteps 是 `sub_1F30A` 退場迴圈的步數（`ebp` 由 0x11 遞減）。
const PhaseBannerExitSteps = PhaseBannerSteps - PhaseBannerEnterSteps

// PhaseBannerStepTickHz 是每一步停留時間的來源：`sub_17AA9(1)` 輪詢 BIOS tick
// 計數 0000:046C 直到差值 ≥ 1，所以每步固定一個 BIOS tick。
const PhaseBannerStepTickHz = 18.2065097

// PhaseBannerStepMillis 是一個 BIOS tick 的毫秒數（約 54.925）。
//
// 這個節奏不依賴 CPU 速度，前提是畫一步的時間小於一個 tick。dosgolem 上這個
// 前提不成立（虛擬 CPU 每指令 1 微秒，畫一步約 356,000 指令），所以收據裡的
// 步距是模擬器工作量，不是原版節奏；節奏寫在 `sub_17AA9` 的參數裡。
const PhaseBannerStepMillis = 1000.0 / PhaseBannerStepTickHz

// PhaseBannerDim 回傳第 step 步要套在 DAC 上的減量。
//
// 原版每一步都呼叫 `sub_11D40(0x10, 0xFF, level)`：索引 0x10..0xFF 的 R／G／B
// 各從主調色盤減去 level 並夾在 0。level 就是迴圈變數（進場 0..15、退場
// 16..0），也就是方塊邊長減一。索引 0x00..0x0F 不動，橫幅字樣因此不變暗。
func PhaseBannerDim(step int) int {
	if step < 0 || step >= PhaseBannerSteps {
		return 0
	}
	return PhaseBannerBlock(step) - 1
}

// PhaseBannerDimRange 是 `sub_11D40` 在橫幅期間套用的 DAC 索引區間。
const (
	PhaseBannerDimFirstIndex = 0x10
	PhaseBannerDimLastIndex  = 0xFF
)

// 字樣滑入與滑出。`sub_1F1CC` 在馬賽克之前跑七次 `sub_1F42D`（參數
// 100／75／50／25／0／1／0），`sub_1F30A` 在重繪地形之後跑五次（0／25／50／
// 75／100）；每一步同樣以 `sub_17AA9(1)` 等一個 BIOS tick。
//
// 每一步畫的是兩塊，而且**相向移動**：第一塊（`PLAYER` 或 `ENEMY`）在
// x ＝ `0x55 − 參數`，第二塊（固定 `PHASE`）在 x ＝ `參數 + 0xA5`，y 都是
// `0x52`（`sub_1F42D` 0x1F43F 的 `mov ebx, 0x55` 與 0x1F473 的 `add eax, 0xa5`）。
// 那是離屏緩衝區的座標，`sub_11EB0` 再整塊搬到 VGA 的 (4,4)，所以換算成 VGA
// 就是停住時的 (0x59,0x56) 與 (0xA9,0x56)——與馬賽克期間那兩塊同一個位置。
// 量測見 docs/knowledge-base/105-phase-banner-timing-20260910.md。
var (
	phaseBannerSlideInArgs  = [...]int{100, 75, 50, 25, 0, 1, 0}
	phaseBannerSlideOutArgs = [...]int{0, 25, 50, 75, 100}
)

// PhaseBannerSlideInSteps 與 PhaseBannerSlideOutSteps 是兩段滑動的步數。
const (
	PhaseBannerSlideInSteps  = len(phaseBannerSlideInArgs)
	PhaseBannerSlideOutSteps = len(phaseBannerSlideOutArgs)
)

// PhaseBannerSlideOriginX 是字樣停住時的原版 x（`0x55`）。滑動的每一步都相對
// 它偏移，偏移量就是 `sub_1F42D` 的參數。
const PhaseBannerSlideOriginX = 0x55

// PhaseBannerSlideInOffset 回傳滑入第 step 步**第一塊**相對停住位置的 x 偏移
// （負值代表還在左邊畫面外）。第二塊取相反數。界外回 0，也就是停住的位置。
func PhaseBannerSlideInOffset(step int) int {
	if step < 0 || step >= PhaseBannerSlideInSteps {
		return 0
	}
	return -phaseBannerSlideInArgs[step]
}

// PhaseBannerSlideOutOffset 回傳滑出第 step 步的 x 偏移。
func PhaseBannerSlideOutOffset(step int) int {
	if step < 0 || step >= PhaseBannerSlideOutSteps {
		return 0
	}
	return -phaseBannerSlideOutArgs[step]
}

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
