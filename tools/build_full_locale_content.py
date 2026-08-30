#!/usr/bin/env python3
"""由 FD2 玩家可見字串清冊產生四語全量內容目錄。

繁中是來源；簡中由 OpenCC 產生可重現初稿；英文使用 Docker 映像內固定的
Argos 離線模型；日文正式產生流程使用固定 revision 的 NLLB。產物明確標記
machine_draft，不冒稱人工校譯。
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
from pathlib import Path

PLACEHOLDER_RE = re.compile(r"%(?:%|[-+0-9.#]*[a-zA-Z])")
ASCII_TOKEN_RE = re.compile(r"[A-Za-z][A-Za-z0-9_.+-]*")
NLLB_MODEL = "facebook/nllb-200-distilled-600M"
NLLB_REVISION = "f8d333a098d19b4fd9a8b18f94170487ad3f821d"


def read_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, value) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def selected_entries(inventory: dict, review: dict) -> list[dict]:
    reviewed_visible = set(review["dispositions"]["player_visible"]["string_ids"])
    excluded = set()
    for name, group in review["dispositions"].items():
        if name != "player_visible":
            excluded.update(group["string_ids"])
    result = []
    for entry in inventory["entries"]:
        if entry["role"] == "go_review":
            if entry["string_id"] in excluded:
                continue
            if entry["string_id"] not in reviewed_visible:
                raise ValueError(f"未審查的 go_review 字串：{entry['string_id']}")
        result.append(entry)
    return result


def canonical_story_ids(entries: list[dict], root: Path) -> dict[str, str]:
    """把 legacy story JSON pointer 對應到已版本化的 document/scene/line ID。"""
    result: dict[str, str] = {}
    documents: dict[str, dict] = {}
    for entry in entries:
        source = entry["source"]
        file = source.get("file", "")
        if not file.startswith("remake/assets/story/") or "json_pointer" not in source:
            continue
        name = Path(file).name
        if name not in documents:
            documents[name] = read_json(root / name)
        document = documents[name]
        pointer = source["json_pointer"]
        if pointer == "/title":
            result[entry["string_id"]] = document["document_id"] + "/title"
            continue
        if pointer == "/location":
            result[entry["string_id"]] = document["document_id"] + "/location"
            continue
        match = re.fullmatch(r"/scenes/(\d+)/(label|lines/(\d+)/(text|speaker_name))", pointer)
        if not match:
            continue
        scene = document["scenes"][int(match.group(1))]
        if match.group(2) == "label":
            result[entry["string_id"]] = scene["scene_id"] + "/label"
        else:
            line = scene["lines"][int(match.group(3))]
            suffix = "text" if match.group(4) == "text" else "speaker-name"
            result[entry["string_id"]] = line["line_id"] + "/" + suffix
    return result


def translate_guarded(text: str, terms: list[tuple[str, str]], convert) -> str:
    """只把普通文字片段送入模型；變數與專名從不進入模型。"""
    replacements = {source: target for source, target in terms}
    protected = [re.escape(source) for source, _ in sorted(terms, key=lambda pair: len(pair[0]), reverse=True)]
    protected.append(r"%(?:%|[-+0-9.#]*[a-zA-Z])")
    protected.append(r"[A-Za-z][A-Za-z0-9_.+-]*")
    pattern = re.compile("(" + "|".join(protected) + ")")
    parts = pattern.split(text)
    result: list[str] = []
    for part in parts:
        if not part:
            continue
        if PLACEHOLDER_RE.fullmatch(part):
            result.append(part)
        elif ASCII_TOKEN_RE.fullmatch(part):
            result.append(part)
        elif part in replacements:
            result.append(replacements[part])
        elif part.strip():
            result.append(convert(part))
        else:
            result.append(part)
    return "".join(result)


def translate_many_nllb(sources: list[str], terms: list[tuple[str, str]], batch_size: int = 16) -> dict[str, str]:
    """以固定 NLLB revision 將受保護片段批次轉成日文。"""
    from transformers import AutoModelForSeq2SeqLM, AutoTokenizer

    replacements = {source: target for source, target in terms}
    protected = [re.escape(source) for source, _ in sorted(terms, key=lambda pair: len(pair[0]), reverse=True)]
    protected.extend((r"%(?:%|[-+0-9.#]*[a-zA-Z])", r"[A-Za-z][A-Za-z0-9_.+-]*"))
    pattern = re.compile("(" + "|".join(protected) + ")")
    layouts: dict[str, list[tuple[str, bool]]] = {}
    fragments: list[str] = []
    for source in sources:
        layout = []
        for part in pattern.split(source):
            if not part:
                continue
            if PLACEHOLDER_RE.fullmatch(part) or ASCII_TOKEN_RE.fullmatch(part):
                layout.append((part, False))
            elif part in replacements:
                layout.append((replacements[part], False))
            elif part.strip():
                layout.append((part, True))
                fragments.append(part)
            else:
                layout.append((part, False))
        layouts[source] = layout
    unique = list(dict.fromkeys(fragments))
    tokenizer = AutoTokenizer.from_pretrained(NLLB_MODEL, revision=NLLB_REVISION, src_lang="zho_Hant")
    translated: dict[str, str] = {}
    ct2_path = os.environ.get("FD2_NLLB_CT2")
    if ct2_path:
        import ctranslate2

        runtime = ctranslate2.Translator(ct2_path, device="cpu", compute_type="int8")
        for start in range(0, len(unique), batch_size):
            batch = unique[start:start + batch_size]
            source_tokens = [tokenizer.convert_ids_to_tokens(tokenizer.encode(text)) for text in batch]
            results = runtime.translate_batch(source_tokens, target_prefix=[["jpn_Jpan"] for _ in batch],
                                              beam_size=2, repetition_penalty=1.2,
                                              no_repeat_ngram_size=3, max_decoding_length=128)
            values = [tokenizer.decode(tokenizer.convert_tokens_to_ids(result.hypotheses[0]),
                                       skip_special_tokens=True) for result in results]
            translated.update(zip(batch, values))
            print(f"ja/NLLB-CT2 fragments: {min(start + batch_size, len(unique))}/{len(unique)}", flush=True)
    else:
        model = AutoModelForSeq2SeqLM.from_pretrained(NLLB_MODEL, revision=NLLB_REVISION)
        target = tokenizer.convert_tokens_to_ids("jpn_Jpan")
        for start in range(0, len(unique), batch_size):
            batch = unique[start:start + batch_size]
            encoded = tokenizer(batch, return_tensors="pt", padding=True, truncation=True, max_length=512)
            generated = model.generate(**encoded, forced_bos_token_id=target, max_new_tokens=256)
            values = tokenizer.batch_decode(generated, skip_special_tokens=True)
            translated.update(zip(batch, values))
            print(f"ja/NLLB fragments: {min(start + batch_size, len(unique))}/{len(unique)}", flush=True)
    return {source: "".join(translated[value] if convert else value for value, convert in layout)
            for source, layout in layouts.items()}


def translator(locale: str):
    if locale == "zh-Hant":
        return lambda text: text, "source"
    if locale == "zh-Hans":
        from opencc import OpenCC

        convert = OpenCC("t2s").convert
        return convert, "machine_draft"
    from argostranslate import translate

    if locale == "en":
        return lambda text: translate.translate(text, "zt", "en"), "machine_draft"
    if locale == "ja":
        return lambda text: translate.translate(translate.translate(text, "zt", "en"), "en", "ja"), "machine_draft"
    raise ValueError(f"不支援的語系：{locale}")


def build(locale: str, entries: list[dict], inventory_path: Path, glossary_path: Path,
          cache_path: Path | None, pivot_path: Path | None, canonical_root: Path,
          cache_only: bool = False, shard_index: int = 0, shard_count: int = 1,
          engine: str = "argos") -> dict:
    if locale == "ja" and engine == "nllb":
        translate_text, status = None, "machine_draft"
    else:
        translate_text, status = translator(locale)
    glossary = read_json(glossary_path)
    terms = [(term["zh-Hant"], term[locale]) for term in glossary["terms"]]
    pivot = None
    if pivot_path:
        pivot_data = read_json(pivot_path)
        pivot = {entry["string_id"]: entry["text"] for entry in pivot_data["entries"]}
        if locale != "ja" or pivot_data["locale"] != "en":
            raise ValueError("--pivot-content 只能供日文使用英文內容目錄")
        terms = [(term["en"], term["ja"]) for term in glossary["terms"]]
        from argostranslate import translate
        translate_text = lambda text: translate.translate(text, "en", "ja")
    cache: dict[str, str] = read_json(cache_path) if cache_path and cache_path.exists() else {}
    output = []
    stable_ids = canonical_story_ids(entries, canonical_root)
    seen_ids: set[str] = set()
    if locale == "ja" and engine == "nllb":
        if pivot_path:
            raise ValueError("NLLB 日文直譯不接受英文中介目錄")
        wanted = []
        for entry in entries:
            source = entry["text"]
            if cache_only and int(hashlib.sha256(source.encode("utf-8")).hexdigest(), 16) % shard_count != shard_index:
                continue
            if source not in cache and source.strip():
                wanted.append(source)
        cache.update(translate_many_nllb(list(dict.fromkeys(wanted)), terms))
    for index, entry in enumerate(entries):
        string_id = stable_ids.get(entry["string_id"], entry["string_id"])
        source = pivot[string_id] if pivot is not None else entry["text"]
        if cache_only and int(hashlib.sha256(source.encode("utf-8")).hexdigest(), 16) % shard_count != shard_index:
            continue
        if source not in cache:
            if not source.strip():
                candidate = source
            elif locale == "ja" and engine == "nllb":
                raise ValueError(f"NLLB 批次快取遺漏：{entry['string_id']}")
            else:
                candidate = translate_guarded(source, terms, translate_text)
            if PLACEHOLDER_RE.findall(candidate) != entry["variables"]:
                raise ValueError(f"變數簽章改變：{entry['string_id']} candidate={candidate!r}")
            if source.strip() and not candidate.strip():
                raise ValueError(f"空譯文：{entry['string_id']}")
            cache[source] = candidate
        if string_id in seen_ids:
            raise ValueError(f"重複翻譯身分：{string_id}")
        seen_ids.add(string_id)
        if not cache_only:
            output.append({
                "string_id": string_id,
                "id_status": "stable_canonical" if entry["string_id"] in stable_ids else entry["id_status"],
                "role": entry["role"],
                "text": cache[source],
                "variables": entry["variables"],
                "status": status,
                "source": entry["source"],
            })
        if index and index % 100 == 0:
            if cache_path:
                write_json(cache_path, cache)
            print(f"{locale}: {index}/{len(entries)}", flush=True)
    if cache_path:
        write_json(cache_path, cache)
    return {
        "schema_version": 1,
        "kind": "fd2_full_locale_content",
        "locale": locale,
        "source_locale": "zh-Hant",
        "entry_count": len(output),
        "inventory_sha256": digest(inventory_path),
        "glossary_sha256": digest(glossary_path),
        "pivot_content_sha256": digest(pivot_path) if pivot_path else None,
        "translation_engine": {
            "zh-Hant": "source",
            "zh-Hans": "OpenCC t2s / opencc-python-reimplemented 0.1.7",
            "en": "Argos translate-zt_en-1_9 / OPUS-MT",
            "ja": (f"NLLB {NLLB_MODEL}@{NLLB_REVISION} zho_Hant->jpn_Jpan"
                   if engine == "nllb" else "Argos translate-zt_en-1_9 then en_ja / OPUS-MT"),
        }[locale],
        "entries": output,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--inventory", type=Path, required=True)
    parser.add_argument("--review", type=Path, required=True)
    parser.add_argument("--glossary", type=Path, required=True)
    parser.add_argument("--locale", choices=("zh-Hant", "zh-Hans", "ja", "en"), required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--cache", type=Path)
    parser.add_argument("--pivot-content", type=Path)
    parser.add_argument("--canonical-root", type=Path, required=True)
    parser.add_argument("--cache-only", action="store_true")
    parser.add_argument("--shard-index", type=int, default=0)
    parser.add_argument("--shard-count", type=int, default=1)
    parser.add_argument("--engine", choices=("argos", "nllb"), default="argos")
    args = parser.parse_args()
    if args.shard_count < 1 or args.shard_index < 0 or args.shard_index >= args.shard_count:
        raise SystemExit("無效的快取分片")
    if args.cache_only and not args.cache:
        raise SystemExit("--cache-only 必須同時指定 --cache")
    inventory = read_json(args.inventory)
    review = read_json(args.review)
    if review["inventory_sha256"] != digest(args.inventory):
        raise SystemExit("字串清冊與人工審查雜湊不一致")
    result = build(args.locale, selected_entries(inventory, review), args.inventory, args.glossary,
                   args.cache, args.pivot_content, args.canonical_root,
                   args.cache_only, args.shard_index, args.shard_count, args.engine)
    if args.cache_only:
        print(f"{args.locale}: warmed cache shard {args.shard_index + 1}/{args.shard_count}")
        return
    write_json(args.output, result)
    if args.cache and args.cache.exists():
        args.cache.unlink()
    print(f"{args.locale}: wrote {result['entry_count']} entries to {args.output}")


if __name__ == "__main__":
    main()
