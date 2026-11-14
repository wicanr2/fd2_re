package main

// Go 名稱轉 Rust 名稱（沿用 R2、R3 骨架產生器的規則）。

import (
	"strings"
	"unicode"
)

var rustKeywords = map[string]bool{
	"as": true, "break": true, "const": true, "continue": true, "crate": true, "else": true, "enum": true,
	"extern": true, "false": true, "fn": true, "for": true, "if": true, "impl": true, "in": true, "let": true,
	"loop": true, "match": true, "mod": true, "move": true, "mut": true, "pub": true, "ref": true,
	"return": true, "self": true, "Self": true, "static": true, "struct": true, "super": true, "trait": true,
	"true": true, "type": true, "unsafe": true, "use": true, "where": true, "while": true, "async": true,
	"await": true, "dyn": true, "abstract": true, "become": true, "box": true, "do": true, "final": true,
	"macro": true, "override": true, "priv": true, "typeof": true, "unsized": true, "virtual": true,
	"yield": true, "try": true, "gen": true,
}

func escapeIdent(s string) string {
	if s == "self" || s == "super" || s == "crate" || s == "Self" {
		return s + "_"
	}
	if rustKeywords[s] {
		return "r#" + s
	}
	return s
}

// snake 把 Go 的 camelCase／PascalCase 轉成 snake_case；連續大寫視為縮寫（HDMap → hd_map）。
func snake(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		if r == '_' {
			b.WriteRune('_')
			continue
		}
		if unicode.IsUpper(r) {
			if i > 0 && rs[i-1] != '_' {
				prev := rs[i-1]
				nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
				// 複數縮寫（IDs）：結尾的 s 跟著縮寫
				if nextLower && rs[i+1] == 's' && (i+2 == len(rs) || !unicode.IsLower(rs[i+2])) && unicode.IsUpper(prev) {
					nextLower = false
				}
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return strings.Trim(out, "_")
}

func screaming(s string) string { return strings.ToUpper(snake(s)) }

// camel 轉成 Rust 型別名稱（首字大寫，去掉底線）。
func camel(s string) string {
	rs := []rune(s)
	if len(rs) == 0 {
		return s
	}
	rs[0] = unicode.ToUpper(rs[0])
	var b strings.Builder
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r == '_' {
			prevLetter := i > 0 && unicode.IsLetter(rs[i-1])
			nextLetter := i+1 < len(rs) && unicode.IsLetter(rs[i+1])
			if prevLetter || nextLetter {
				if i+1 < len(rs) {
					rs[i+1] = unicode.ToUpper(rs[i+1])
				}
				continue
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}
