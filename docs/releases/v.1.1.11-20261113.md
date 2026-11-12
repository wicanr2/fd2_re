# v.1.1.11-20261113 發行說明

本版修正 Windows 完整版回報的戰中存檔問題，並在每次執行時留下執行紀錄，供當機回報使用。三平台完整版以新引擎重新打包。素材只有四份語系檔加入兩則存檔提示，其餘與 `v.1.1.9-20261111` 相同。

## 本次修正

| 項目 | 修正內容 |
|---|---|
| 戰鬥中 F5 快存 | 我方待機時保存整個戰場，包含單位位置、HP、回合、已觸發事件、寶箱與亂數狀態。F9 回到同一回合，不再回到章首。 |
| 戰鬥選單存檔 | 沒有原版存檔基準時改存同一份戰場快照，不再出現存檔失敗。 |
| 不能存檔的時機 | 敵方回合、劇情、對話、移動或攻擊演出中按 F5，畫面提示稍後再存，不寫入檔案。 |
| 執行紀錄 | 存檔資料夾內寫入 `fd2-run.log`，前一次執行保留為 `fd2-run.prev.log`。紀錄錯誤訊息、狀態提示，以及程式崩潰時的錯誤堆疊。 |
| 存檔位置 | 家目錄的 `.local/share/fd2_re`；有設定 `XDG_DATA_HOME` 時改用該目錄下的 `fd2_re`。 |

戰場快照是重製自有的 JSON 格式，存在重製存檔內，不寫入原版 `FD2.SAV` 的章節槽。舊版存檔可繼續讀取。規格與驗證見[系統設計第 56 篇「#303 戰場JSON快照」](../knowledge-base/56-fd2-remake-sdd.md)。

## 仍在追查

| 工單 | 狀態 |
|---|---|
| [#303](https://github.com/wicanr2/fd2_re/issues/303) 戰中存檔 | 已修正，重製端測試通過，等 Windows 實機確認後結案。 |
| [#304](https://github.com/wicanr2/fd2_re/issues/304) 當機 | 根因未知。本版加上執行紀錄，請在當機後附上 `fd2-run.log` 與 `fd2-run.prev.log`。 |
| [#305](https://github.com/wicanr2/fd2_re/issues/305) HD 畫面閃爍 | Linux OpenGL 無法重現。需要 Windows 錄影與工具列顯示的繪圖後端及幀率。 |

## 交付內容

| 交付內容 | 位置 |
|---|---|
| Windows、macOS 通用版、Linux 本機完整版 | `dist-all/v.1.1.11-20261113/full-local/` |
| 三平台引擎包 | `dist-all/v.1.1.11-20261113/patch/` |
| 推廣影片 | 沿用 `dist-all/v.1.1.10-20261112/promo/` |
| 雜湊清單與驗收收據 | `dist-all/v.1.1.11-20261113/SHA256SUMS.json`、`smoke/` |

| 完整版 | 位元組 | SHA-256 |
|---|---|---|
| Windows x86_64 ZIP | 3748400365 | `49ed9056c44c3b6a2100162a45fd0c95843a6af0b3c83db07c1a763789f1faf6` |
| macOS 通用版 ZIP | 3757084494 | `985ae7d682318ba1181cfbe06a1a197b22956ec6eefedec7a3c910968385e626` |
| Linux x86_64 AppImage | 3236526584 | `7411a96a5641b0fda965c27a95fcf76cbbddc1affa2ad1274a9bda3dd033d410` |

完整版含原版衍生素材，只供合法持有者本機使用，不公開上傳。本版沒有建立 tag 或公開 Release。

## 建置與驗證範圍

三平台引擎由來源提交 `5186e022` 在本機 Docker 建置，不經 GitHub Actions。方式見[打包文件 §4.4](../knowledge-base/41-packaging.md#44-本機-osxcross-建置使用者2026-11-13定案)。

| 平台 | 建置 | 已驗證 |
|---|---|---|
| Linux | 與 CI 同一支 `build-appimage.sh` | 引擎包與完整包在 Xvfb 下自檢；完整包開場受控截圖 |
| Windows | mingw-w64 交叉建置 | 引擎包與完整包在 Wine 10 下自檢；完整包開場受控截圖 |
| macOS 通用版 | osxcross SDK 15.5，arm64 與 x86_64 各編再合併 | 只做靜態檢查：雙架構、arm64 簽章、只連結系統框架、最低 macOS 11.0 |

三包各 54577 份素材已獨立解包逐檔核對。與 v.1.1.9 完整版相比，只有四份語系檔 `pack.json` 不同，HD 素材與原版分離素材完全相同。Linux 與 Wine 的開場截圖逐位元相同，也與 v.1.1.9 的截圖相同。

macOS 版在 Linux 上無法執行，沒有任何執行證據；Windows 的自檢在 Wine 執行，不等於原生 Windows。Windows、macOS 未簽署，macOS 未公證，首次開啟需右鍵選「打開」。戰中存檔的修正仍需在 Windows 實機確認。
