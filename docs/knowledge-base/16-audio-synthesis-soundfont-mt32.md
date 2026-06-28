# 16 — 音色合成:SoundFont、MT-32 與版本切換

> 評估「炎龍騎士團2 的音樂能不能提供多種音源版本(SoundFont / MT-32),怎麼切換」,
> 並解釋 **SoundFont 到底是什麼**。音樂資料格式見 `07`(XMIDI)、播放機制見 `12`。

## 先建立觀念:MIDI ≠ 聲音

MIDI(以及 FD2 用的 XMIDI)**不是錄音,是「演奏指令」**——「第 3 軌、用第 48 號樂器、彈中央 C、力度 90、持續半拍」。
真正發出聲音的是**音源(synthesizer)**。同一份 MIDI,換不同音源 = 完全不同的音色。1990 年代 DOS 遊戲的三大音源:

| 音源 | 怎麼發聲 | 音色特徵 |
|---|---|---|
| **OPL2/3 FM**(AdLib / Sound Blaster) | 晶片用 FM 合成即時算波形 | 金屬感、復古「電子」味,最普及最便宜 |
| **Roland MT-32 / CM-32L** | 外接音源模組,內含取樣 + LA 合成 | 厚實、擬真,當年高階玩家的「天花板」 |
| **General MIDI(GM)+ 取樣** | 標準 128 樂器表 + 取樣音色 | 後來的標準;音色依音源庫而定 |

**FD2 對三者都支援**(隨遊戲附整套 `.MDI` 驅動,玩家用 `SETSOUND.EXE` 選)。**關鍵**:既然附了
`MT32MPU.MDI`,代表**音樂本來就為 MT-32 編寫**——MT-32 版不是「腦補」,是還原原意。

## `.MDI` 是什麼?`MT32MPU.MDI` 又是幹嘛的?

`.MDI` = **Miles AIL 的「音樂裝置驅動」**,一個檔案對應一種音源/音效卡。遊戲啟動時依玩家設定載入對應 `.MDI`,
AIL 就知道怎麼把 XMI 音樂送到那個裝置發聲(見 `12`)。FD2 附的 `.MDI`(檔內都有自我描述字串):

| 驅動 | 內部描述(EXE/檔內字串) | 對應硬體 |
|---|---|---|
| `ADLIB.MDI` / `ADLIBG.MDI` | AdLib | AdLib FM(OPL2) |
| `OPL3.MDI` | OPL3 | Sound Blaster Pro/16 的 OPL3 FM |
| `SBLASTER.MDI` / `SBPRO1/2.MDI` | Sound Blaster 系列 | 聲霸卡 FM |
| `SBAWE32.MDI` | Sound Blaster AWE32 | AWE32 的 **EMU8000 取樣合成(SoundFont 之源)** |
| **`MT32MPU.MDI`** | **"Roland MT-32 MIDI with MPU-401 MIDI Interface" / "Roland LAPC-1"** | **真・Roland MT-32 / CM-32L / LAPC-I** |
| `MPU401.MDI` | "General MIDI (Roland MPU-401 interface or 100% compatible)" | 一般 General MIDI 裝置 |
| `ULTRA.MDI` | Gravis UltraSound | GUS 取樣合成 |
| `PAS.MDI`/`PASPLUS.MDI` | Pro AudioSpectrum | PAS 卡 |
| `TANDY.MDI` / `PCSPKR.MDI` | Tandy 3-voice / PC 喇叭 | 低階內建發聲 |
| `NULL.MDI` | (無聲) | 關閉音樂 |

**`MT32MPU.MDI` 拆開看**:
- **MT32** = 目標音源是 **Roland MT-32**(及其超集 CM-32L / 卡式版 LAPC-I)。
- **MPU**(MPU-401)= 當年 PC 送 MIDI 出去的**標準介面卡**(Roland 的 MIDI Processing Unit)。
  電腦本身不會 MT-32 的聲音——它只是透過 MPU-401 這個「MIDI 輸出埠」,把音符指令送給**外接的 MT-32 硬體模組**去發聲。
- 所以 `MT32MPU.MDI` 的意思就是:**「把音樂用 MT-32 的音色設定,經 MPU-401 埠送到外接的 Roland MT-32 播放」**。

**和 `MPU401.MDI` 差在哪?** 兩者都走 MPU-401 這個埠,但:
- `MT32MPU.MDI` 用 **MT-32 專屬的樂器對映 / SysEx 設定**(MT-32 的內建音色排列和 GM 不同)。
- `MPU401.MDI` 用 **General MIDI 標準對映**(給 GM 相容的音源)。
→ 同一份音樂,FD2 會依你接的是 MT-32 還是 GM 音源,送出對的樂器編號。**FD2 為 MT-32 特別做了驅動 = 它的音樂是按 MT-32 音色調過的**。

> 一句話:`MT32MPU.MDI` = 「透過 MPU-401 介面卡,驅動外接 Roland MT-32 音源」的播放驅動。
> 它的存在,就是「FD2 音樂原本就該用 MT-32 聽」的鐵證。

## 什麼是 SoundFont(.sf2)?

**SoundFont = 一個檔案,裡面裝了「每個樂器的取樣錄音 + 怎麼播放它們的規則」**,讓軟體合成器(如 fluidsynth)
能把 MIDI 變成聲音。可以把它想成「一整套虛擬樂器的音色包」:

- 內容:各樂器在不同音高的**取樣波形**(PCM)、迴圈點、音量包絡(ADSR)、濾波、力度分層、key range…
- 標準:Creative 在 1990s 為 AWE32 音效卡推出(`SBAWE32.MDI` 就是 FD2 的 AWE32 驅動);`.sf2` 成為通用格式。
- **GM SoundFont**:依 General MIDI 128 樂器表收音色(鋼琴=0、小提琴=40…),任何 GM SoundFont 都能播任何 GM MIDI,
  音色好壞看 SoundFont 品質(從幾 MB 的小包到數百 MB 的管弦樂包)。
- **與 MT-32 的差別**:GM SoundFont 是「通用 MIDI 音色」,**不是 MT-32 的音色**——同一首歌用 GM SoundFont 播,
  樂器對得上但「聽起來不像當年的 MT-32」。要道地 MT-32,得用**真正的 MT-32 ROM 韌體**透過模擬器發聲(見下)。

> 一句話:**SoundFont 是「軟體用的取樣音色包」;MT-32 是「特定硬體的音色」**。兩者都能播 FD2 的 MIDI,音色不同。

## MT-32 版可行嗎?——可以,**已實證**(item 2)

**結論:高度可行,且比多數遊戲更名正言順**(因為 FD2 原生支援 MT-32)。**已實際渲染全 15 首成功**。管線:

```
FD2 XMI(FDMUS.DAT)  → tools/xmi2mid.py → 標準 MIDI
                     → munt(MT-32 模擬器)+ 真 MT-32 ROM → WAV(道地 MT-32 音色)
```

**實證結果**:`tools/export_mt32.sh` 已把 15 首 FDMUS 全數渲染成 Roland MT-32 WAV
(例:FDMUS_008 = 155 秒、32kHz 立體聲),輸出於本機 `extracted/music_mt32/`。完全可行。

- **munt** 是 DOSBox/ScummVM 內建的 MT-32 模擬器,用**真正的 Roland ROM**(CONTROL + PCM)發聲 = 道地音色。
  以 `mt32emu-smf2wav` 離線把 MIDI 批次轉 WAV(跑完即止,不會 real-time 串流暴衝)。
- **ROM**:本機已有(`/home/anr2/cht/mt32/`):經典 MT-32(`MT32_CONTROL` 64KB + `MT32_PCM` 512KB)與
  CM-32L(超集,音色更多)。⚠ ROM 是 Roland 版權韌體,**只在本機、不入庫、不散布**(從自有實機 dump 的保存圈做法)。
- **建置**:munt 用 Docker 編(Debian + cmake;先 build `mt32emu` 庫再 build `mt32emu_smf2wav`)。
  可直接沿用既有做法(參考 `~/dq3/docs/59-munt-mt32-build.md` 與 `~/dq3/tools/export_mt32_wav.sh`)。
- **音色對映**:XMI 的 `TIMB` chunk + program change 帶樂器指定;FD2 既為 MT-32 編寫,直接送 munt 即正確發聲
  (若部分曲用 GM 號,munt 也能播,音色為 MT-32 對應樂器)。

## 版本切換:SoundFont / MT-32 /(OPL)怎麼共存(item 3)

重製版可提供「音源」選單,三種音色都從**同一份 XMI→MIDI**產生:

| 版本 | 產生方式 | 重製端做法 |
|---|---|---|
| **MT-32**(道地原味) | munt + MT-32 ROM 離線 render → WAV | 內建 WAV 串流播放(零解碼依賴);assets 放 `mt32/track_NN.wav` |
| **SoundFont(GM)** | 兩條路:① 執行期 fluidsynth + 玩家自選 `.sf2`(可換音色包);② 預先 render WAV | C++ 用 fluidsynth/tsf;Go/Ebiten 用 oto + tinysoundfont |
| **OPL/FM(復古)** | 模擬 OPL3 播 MIDI,或保留原始 FM | 可選,最還原「聲霸卡」聽感 |

**建議架構**:
- 設定選單「音源:MT-32 / SoundFont / FM」。
- **MT-32** 用「預先 render 的 WAV」(ROM 不能散布,但玩家可自備 ROM 自行 render;或提供腳本)。
- **SoundFont** 用「執行期合成 + 可換 .sf2」(彈性、檔案小,玩家可換喜歡的音色包)。
- 兩者共用 `xmi2mid.py` 轉出的標準 MIDI;切換 = 換播放後端。

> 對「保存原味」而言:**MT-32 版最接近 1995 年高階玩家聽到的聲音**(因 FD2 為其編寫),值得做為旗艦音源;
> SoundFont 版提供彈性與低門檻;FM 版還原最普及的聲霸卡聽感。三者並存,玩家自選。

## 工具
- `tools/export_mt32.sh <raw/FDMUS> <ROM目錄> <out>`:XMI→MIDI→munt 全自動渲染 MT-32 WAV。
- `tools/xmi2mid.py`:XMI→標準 MIDI(SoundFont / 其他合成器共用)。

## 待辦 / 下一步
- ✅ munt + MT-32 ROM 渲染 15 首 → 本機 `extracted/music_mt32/`(已完成)。
- 試聽校對 + 比對 CM-32L(超集音色)版本;挑旗艦音源。
- 接 SoundFont(GM)版:選一套 `.sf2`,render 或執行期合成,與 MT-32 版 A/B 比較。
- XMI `TIMB`→ 樂器對映表整理(各曲配器),供 SoundFont 版微調。
