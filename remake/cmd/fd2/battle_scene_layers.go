package main

// battleSceneLayer 是全螢幕戰鬥演出的三個圖層。
type battleSceneLayer int

const (
	battleLayerDefenderFigure battleSceneLayer = iota // 守方 figure（待機幀循環）
	battleLayerOwnPedestal                            // 我方腳下的 TAI 台座
	battleLayerAttackerFigure                         // 攻方 figure（攻擊幀序）
)

// battleSceneLayerOrder 回傳三個圖層的繪製順序。
//
// 台座跟隨的是**我方**，不是當回合的攻方或守方：doc35 §3.2.5 已把「用攻方／
// 守方描述左右」正名為誤框架——我方是背影在右並踩台座，敵方是正面在左且沒有
// 台座。反組譯側 `0x29164` 在 `0x28c46` 載入 TAI 之後與 figure 一起畫在腳下。
// 原版收據（docs/data/ui-traces/fd2-physical-attack-presentation-20260909.json）
// 拍到敵方攻擊我方那一側，我方仍在右邊、踩在台座上、腳沒有被蓋住。
//
// 左右是資料決定的，不是角色扮演的角色決定的：FIGANI 幀標頭內嵌的絕對座標按
// 陣營分邊，我方亞雷斯（資源 12／13）待機與攻擊都在 x=89..178，敵方盜賊
// （資源 288／289）兩者都在 x=6..28。同一單位不論攻守都留在自己那一側，因此
// 唯一會壓到台座的是我方那張圖。
//
// 所以台座必須緊接在我方 figure 之前：我方攻擊時我方是攻方，敵方攻擊時我方是
// 守方，順序跟著換。
func battleSceneLayerOrder(atkOwn bool) [3]battleSceneLayer {
	if atkOwn {
		return [3]battleSceneLayer{
			battleLayerDefenderFigure, battleLayerOwnPedestal, battleLayerAttackerFigure,
		}
	}
	return [3]battleSceneLayer{
		battleLayerAttackerFigure, battleLayerOwnPedestal, battleLayerDefenderFigure,
	}
}
