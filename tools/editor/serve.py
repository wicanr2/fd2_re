#!/usr/bin/env python3
"""編輯器的本機檔案橋。

兩個用途：

1. **跨瀏覽器**。File System Access API 只有 Chrome 與 Edge 有；Firefox 與 Safari
   開得了編輯器卻存不了檔。doc 38 §4 早就把這支列為退路。
2. **讓來回可以被驗證**。授權目錄需要使用者手勢，所以「開啟→改→存回→引擎讀得
   起來」這條路只有人走得完。走 HTTP 之後機器也走得完，那一條才驗得起來。

只做兩件事：服務 `tools/editor/` 的靜態頁面，以及在白名單路徑上讀寫 JSON。

安全邊界（都不可放寬）：

- 只綁 `127.0.0.1`。
- 讀寫都限制在 `remake/assets/` 底下；解析後的真實路徑要仍在那個根目錄內，符號連結
  指出去一律拒絕。**寫入只允許 `.json`**；`.png` 只讀不寫，戰場編輯器要拿 tileset。
- 寫入前先驗 JSON 解析得動——寫進半份壞檔會讓遊戲在啟動時才失敗。
- **沿用原檔的縮排**。受版控的 JSON 縮排不一致（story 用 1 空格、scenario 用 2、
  map.json 是單行），統一格式會讓每次存檔都產生整份 diff，把真正改到的那一行埋掉。
- 不提供刪除或建立新檔：編輯器改的是既有資料。
- **保不住格式就拒絕寫。** 有些受版控的 JSON 是手工排版的（`cutscenes/acting/*.json`
  把 `{ "slot": 34, "pose": 2 }` 寫在同一行），程式化的 dump 重現不了。寫回等於整份
  重排，真正改到的那一行就埋在幾百行的 diff 裡。寫入前先確認「原封不動 dump 回去
  等於原檔」，不成立就回 409。

用法：
    python3 tools/editor/serve.py [--port 8765]
    瀏覽器開 http://127.0.0.1:8765/battlefield.html
"""

import argparse
import http.server
import json
import pathlib
import posixpath
import sys
import urllib.parse

ROOT = pathlib.Path(__file__).resolve().parent.parent.parent
EDITOR_DIR = ROOT / "tools" / "editor"
ALLOWED_ROOT = (ROOT / "remake" / "assets").resolve()
READABLE_SUFFIXES = {".json", ".png"}
WRITABLE_SUFFIXES = {".json"}


def resolve(raw, suffixes):
    """把 API 收到的相對路徑解析成真實檔案；越界或副檔名不合一律回 None。"""
    if not raw:
        return None
    candidate = (ROOT / posixpath.normpath(raw.lstrip("/"))).resolve()
    if candidate.suffix not in suffixes:
        return None
    if not candidate.is_relative_to(ALLOWED_ROOT):
        return None
    return candidate


def style_of(path):
    """偵測原檔的縮排、分隔符與結尾換行。

    三樣都要沿用，否則原樣存回也會產生 diff：

    - 縮排：story 用 1 空格、scenario 用 2、`map.json` 是單行。
    - 分隔符：單行檔是 `{"w":24,"h":24`（完全緊湊），假設成 `, ` 會讓唯一的那一行
      整行變動。
    - 結尾換行：`map0_units.json` 沒有，多補一個就差 1 byte——內容一模一樣，diff
      卻說整份檔案改了。
    """
    text = path.read_text(encoding="utf-8")
    trailing = text.endswith("\n")
    lines = text.splitlines()
    if len(lines) >= 2:
        second = lines[1]
        width = len(second) - len(second.lstrip(" "))
        if width:
            return width, (",", ": "), trailing
    head = text[:400]
    compact = not ('", "' in head or '": "' in head)
    return None, (",", ":") if compact else (", ", ": "), trailing


def dump(value, style):
    indent, separators, trailing = style
    body = json.dumps(value, ensure_ascii=False, indent=indent, separators=separators)
    return body + "\n" if trailing else body


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(EDITOR_DIR), **kwargs)

    def log_message(self, fmt, *args):  # 安靜一點；出事看回應碼就夠
        sys.stderr.write("%s %s\n" % (self.address_string(), fmt % args))

    def _send(self, code, payload):
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        if parsed.path == "/api/list":
            prefix = urllib.parse.parse_qs(parsed.query).get("prefix", [""])[0]
            base = resolve(prefix + "/_.json", READABLE_SUFFIXES)
            if base is None:
                self._send(403, {"error": "路徑不在白名單內"})
                return
            directory = base.parent
            names = sorted(p.name for p in directory.glob("*.json")) if directory.is_dir() else []
            self._send(200, {"prefix": prefix, "files": names})
            return
        if parsed.path == "/api/blob":
            target = resolve(urllib.parse.parse_qs(parsed.query).get("path", [""])[0],
                             {".png"})
            if target is None or not target.is_file():
                self._send(403 if target is None else 404, {"error": "讀不到這個檔"})
                return
            body = target.read_bytes()
            self.send_response(200)
            self.send_header("Content-Type", "image/png")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if parsed.path == "/api/file":
            target = resolve(urllib.parse.parse_qs(parsed.query).get("path", [""])[0],
                             {".json"})
            if target is None:
                self._send(403, {"error": "路徑不在白名單內"})
                return
            if not target.is_file():
                self._send(404, {"error": "找不到檔案"})
                return
            self._send(200, {"path": str(target.relative_to(ROOT)),
                             "content": json.loads(target.read_text(encoding="utf-8"))})
            return
        super().do_GET()

    def do_PUT(self):
        parsed = urllib.parse.urlparse(self.path)
        if parsed.path != "/api/file":
            self._send(404, {"error": "沒有這個端點"})
            return
        target = resolve(urllib.parse.parse_qs(parsed.query).get("path", [""])[0],
                         WRITABLE_SUFFIXES)
        if target is None:
            self._send(403, {"error": "路徑不在白名單內"})
            return
        if not target.is_file():
            # 只改既有資料；不接受建立新檔，免得打錯路徑就多出一個沒人讀的檔案。
            self._send(404, {"error": "只能覆寫既有檔案"})
            return
        # 保不住格式就不要動它。這個檢查只看原檔，與要寫入的內容無關：dump 不回
        # 原樣，表示這份檔案的排版是程式重現不了的。
        original = target.read_text(encoding="utf-8")
        style = style_of(target)
        try:
            faithful = dump(json.loads(original), style) == original
        except json.JSONDecodeError:
            faithful = False
        if not faithful:
            self._send(409, {"error": "這份檔案的排版程式重現不了（多半是手工排版），"
                                      "寫回會整份重排，所以不動它"})
            return
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length).decode("utf-8")
        try:
            value = json.loads(raw)
        except json.JSONDecodeError as err:
            self._send(400, {"error": f"不是合法 JSON：{err}"})
            return
        target.write_text(dump(value, style), encoding="utf-8")
        self._send(200, {"path": str(target.relative_to(ROOT)), "bytes": target.stat().st_size})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=int, default=8765)
    args = parser.parse_args()
    server = http.server.ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    print(f"編輯器：http://127.0.0.1:{args.port}/battlefield.html")
    print(f"　　　　http://127.0.0.1:{args.port}/campaign.html")
    print(f"可讀：{ALLOWED_ROOT.relative_to(ROOT)}／*.json、*.png")
    print(f"可寫：同上，只有 *.json，而且只覆寫既有檔案")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print()
    return 0


if __name__ == "__main__":
    sys.exit(main())
