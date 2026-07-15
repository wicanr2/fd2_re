# 對白索引映射（可編輯劇本資料）

`count-aligned.json` 把原版 `FDTXT_NNN` 的 offset-table string index 對到
`assets/story/` 的可編輯 scene/line。它只含「原始 logical utterance 總數」與
劇本 line 總數完全一致的映射；`diagnostics` 中的資源尚未映射，使用時必須回報
未解決狀態，不能用相似文字或章號猜測。

可由玩家自備原版資料重建：

```sh
python3 tools/export_story_index_map.py \
  extracted/raw/FDTXT remake/assets/story \
  remake/assets/cutscenes/dialogue-index/count-aligned.json
```

這份資料不儲存原始二進位或額外台詞內容，只保存可審計的索引／scene／line 關係。
同一份 FDTXT 可以在不同劇本 context 重用，所以 resolver 的 key 一律是
`source_dat + script + string_index`，不是 string index 單獨使用。
