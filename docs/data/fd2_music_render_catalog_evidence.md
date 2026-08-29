# FD2 音樂標準素材與音源 catalog 證據

## 證據性質

本檔整理既有音樂資產及 runtime consumer，不新增曲名或作曲語意。原版播放控制流
沿用 [`12-music-playback-and-scene.md`](../knowledge-base/12-music-playback-and-scene.md)
的 `0x25977(track, loop_count)` 證據；XMIDI 結構沿用
[`07-music-xmidi-format.md`](../knowledge-base/07-music-xmidi-format.md)。

固定原版來源：

| 檔案 | bytes | MD5 | SHA-256 |
|---|---:|---|---|
| `FDMUS.DAT` | 80,367 | `4dfa214125edcc4658acbba2e1201a28` | `4105ebde543fe1c497e852728f6bc333bda80edeb7fb3671e487504bee74e998` |
| `MDI.INI` | 218 | `1eb7d3bb2f1f781b306c3a580fe3273d` | `d42a9f90e09585eee203db863c664d5cae763e00749325c91b028ee3bcac55d1` |
| `SAMPLE.AD`／`SAMPLE.OPL` | 3,242 | `b04ce567e1fb40e8c1a2002b23027dfd` | `87ddb683096b504bcaea693f8a84d1d36fd431df0c576eaafcf0bf8e21e3ed5b` |

`FDMUS.DAT` 的有效 XMIDI resource 是 `1,3,4,6,8,10..19`，共15首；現有完整
抽取已把它們轉成相同編號的標準 MIDI。這只證明演奏事件投影，不證明任一預錄
音源的逐樣本輸出。

## 現有 OGG 實檔盤點（2026-08-29）

正式設定只列 `fm` 與 `mt32` 兩個音源，各有15份 Ogg Vorbis、雙聲道、192,000 Hz。
所有檔案只有 encoder comment，沒有 `LOOPSTART`／`LOOPEND` 或其他 loop tag。
`assets/music/` 另有15份44,100 Hz legacy OGG；其生成身分未由檔內 metadata 或
受版控紀錄閉合，因此只能作研究／遷移來源，不能成為正式 fallback 或冒充第三個音源。

FM 版的受控生成意圖是：固定 `FDMUS.DAT` → XMIDI-to-MIDI → 固定
`SAMPLE.AD`／`SAMPLE.OPL` → libADLMIDI → OGG。原版 `MDI.INI` 的預設驅動及遊戲
音色庫支持「Sound Blaster／AdLib FM」身分；但現有 OGG 沒有保存 libADLMIDI commit、
WOPL converter version及 encoder完整版本鏈，所以既有 render 的重生 provenance
只能標為「不完整舊 render」，不能宣稱 bit-identical rebuild。

MT-32 版的受控生成意圖是：固定 `FDMUS.DAT` → XMIDI-to-MIDI → munt + 玩家合法
持有的 Roland MT-32 ROM → OGG。ROM 不入庫、不進 catalog、不隨工具散布。現有 OGG
沒有保存 ROM hash、munt commit及完整命令鏈，所以同樣只能標為「不完整舊 render」。
使用 proprietary ROM 產生的聲音也不自動取得重新授權；catalog 只記錄技術來源，
不宣稱公有領域、免權利或可商用。

## loop／stop 邊界

已證實的原版 ABI 只有 caller 傳入的 `loop_count`：`0` 是持續循環、`1` 是播一次，
負值或大於1不屬目前正式重製契約。現有 OGG 沒有內嵌 loop 點，runtime 以整份 decoded
stream 的 `0..pcm_samples` 重播；這是玩家可聽的 E1 近似，不是無縫波形或 DOS/Miles
逐 tick parity。catalog 必須明列 `whole_file_runtime_repeat`、每個 variant 的 sample
rate／sample count，以及 `seam_evidence=unknown`，不可用檔案存在推導無縫 loop。

## 分離 catalog 邊界

穩定 catalog 必須列出：固定 track ID、原始 FDMUS resource、兩個正式 render
profile、相對 OGG 路徑、bytes、SHA-256、channels、sample rate、PCM sample count、
duration、render provenance、權利註記、loop mode及 seam evidence。runtime 只接受
catalog 內已有且 hash／Vorbis geometry 相符的選定 profile，不再靜默落回未分級
`assets/music/`。缺 catalog、track、profile或檔案時保持不播放，不改變 campaign state。

本資料契約不為15首曲目命名，也不把場景中常見的 track consumer外推成唯一用途。

完整generated pack不內含或複製這30份OGG。它的manifest以
`runtime_catalogs.music`保存catalog bytes／SHA-256與2 profile／15 track／30 render
計數，資產位置只記邏輯根`runtime_assets`；產生及驗證時由呼叫者明確注入實體root。
因此manifest不保存`../`穿越路徑，也不會把15份中間MIDI誤改成已匯出的OGG。看到bridge
但缺外部root、catalog或任一render時，完整pack驗證失敗即關閉。
