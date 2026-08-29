// fd2-string-inventory 產生多語系遷移前的可重跑繁中文字串診斷清冊。
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var stableID = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
var formatVariable = regexp.MustCompile(`%([0-9]+\$)?[-+# 0]*([0-9]+|\*)?(\.[0-9*]+)?[vTtbcdoOqxXUeEfgGswxp]`)
var unsafeIDCharacters = regexp.MustCompile(`[^a-z0-9._/-]+`)

type Source struct {
	File        string `json:"file"`
	JSONPointer string `json:"json_pointer,omitempty"`
	Function    string `json:"function,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
}

type Entry struct {
	StringID   string   `json:"string_id"`
	IDStatus   string   `json:"id_status"`
	Role       string   `json:"role"`
	Confidence string   `json:"confidence"`
	Text       string   `json:"text"`
	Variables  []string `json:"variables"`
	Source     Source   `json:"source"`
}

type Inventory struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Locale        string            `json:"locale"`
	Status        string            `json:"status"`
	GeneratedBy   map[string]string `json:"generated_by"`
	Entries       []Entry           `json:"entries"`
}

type Summary struct {
	SchemaVersion      int               `json:"schema_version"`
	Kind               string            `json:"kind"`
	Locale             string            `json:"locale"`
	Status             string            `json:"status"`
	GeneratedBy        map[string]string `json:"generated_by"`
	InventorySHA256    string            `json:"inventory_sha256"`
	EntryCount         int               `json:"entry_count"`
	UniqueTextCount    int               `json:"unique_text_count"`
	VariableEntryCount int               `json:"variable_entry_count"`
	ByIDStatus         map[string]int    `json:"by_id_status"`
	ByRole             map[string]int    `json:"by_role"`
	ByRoleUniqueText   map[string]int    `json:"by_role_unique_text"`
	ByConfidence       map[string]int    `json:"by_confidence"`
}

var jsonRoles = map[string]map[string]string{
	"story": {
		"text": "dialogue", "speaker_name": "character_name", "label": "scene_label",
		"title": "chapter_title", "location": "location_name",
	},
	"scenarios": {
		"text": "dialogue_or_system", "label": "option_or_ui", "name": "entity_name",
		"title": "chapter_title", "prompt": "prompt", "town": "location_name",
		"speaker_name": "character_name", "cls_name": "class_name",
	},
	"data": {
		"name": "entity_name", "label": "command_or_ui", "title": "title",
		"description": "description",
	},
}

func variables(text string) []string {
	matches := formatVariable.FindAllString(text, -1)
	if matches == nil {
		return []string{}
	}
	return matches
}

func escapePointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func provisionalID(prefix, relative, pointer string) string {
	value := strings.TrimSuffix(filepath.ToSlash(relative), filepath.Ext(relative))
	value = strings.ToLower(value)
	value = unsafeIDCharacters.ReplaceAllString(value, "-")
	pointer = strings.Trim(pointer, "/")
	pointer = unsafeIDCharacters.ReplaceAllString(strings.ToLower(pointer), "-")
	return strings.TrimSuffix(prefix+"."+strings.ReplaceAll(value, "/", ".")+"."+strings.ReplaceAll(pointer, "/", "."), ".")
}

func walkJSON(value any, relative, top, pointer string, entries *[]Entry) {
	switch node := value.(type) {
	case map[string]any:
		translatableFields := 0
		for key, child := range node {
			if text, ok := child.(string); ok && text != "" && jsonRoles[top][key] != "" {
				translatableFields++
			}
		}
		for key, child := range node {
			childPointer := pointer + "/" + escapePointer(key)
			if text, ok := child.(string); ok && text != "" {
				if role := jsonRoles[top][key]; role != "" {
					id := ""
					status := "provisional_legacy_path"
					// 只有單一文字值節點可直接採用既有 string_id。角色名與台詞等多欄
					// 物件若共用同一 ID，會把兩個翻譯單元錯誤合併，故仍以來源鍵候選化。
					if candidate, ok := node["string_id"].(string); translatableFields == 1 && ok && stableID.MatchString(candidate) {
						id, status = candidate, "existing_stable_id"
					}
					if id == "" {
						id = provisionalID("legacy.json", relative, childPointer)
					}
					*entries = append(*entries, Entry{
						StringID: id, IDStatus: status, Role: role, Confidence: "field_rule",
						Text: text, Variables: variables(text),
						Source: Source{File: filepath.ToSlash(relative), JSONPointer: childPointer},
					})
				}
			}
			walkJSON(child, relative, top, childPointer, entries)
		}
	case []any:
		for index, child := range node {
			walkJSON(child, relative, top, fmt.Sprintf("%s/%d", pointer, index), entries)
		}
	}
}

func collectJSON(repo string) ([]Entry, error) {
	root := filepath.Join(repo, "remake", "assets")
	var entries []Entry
	for _, top := range []string{"story", "scenarios", "data"} {
		directory := filepath.Join(root, top)
		err := filepath.WalkDir(directory, func(path string, item os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if item.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var document any
			if err := json.Unmarshal(raw, &document); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			relative, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			walkJSON(document, relative, top, "", &entries)
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return entries, nil
}

func hasNonASCII(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func selectorName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	}
	return ""
}

func goRole(stack []ast.Node) (string, string, bool) {
	directUI := false
	internalOnly := false
	semanticRole := ""
	for _, node := range stack {
		switch value := node.(type) {
		case *ast.FuncDecl:
			name := strings.ToLower(value.Name.Name)
			if strings.Contains(name, "draw") || strings.Contains(name, "render") {
				directUI = true
			}
			if strings.Contains(name, "speakername") || strings.Contains(name, "displayname") || strings.Contains(name, "classname") {
				semanticRole = "entity_name"
			}
		case *ast.CallExpr:
			name := selectorName(value.Fun)
			lowerName := strings.ToLower(name)
			if strings.Contains(lowerName, "draw") || strings.Contains(lowerName, "render") || strings.Contains(lowerName, "wrap") {
				directUI = true
			}
			if name == "Errorf" || name == "New" || strings.HasPrefix(name, "Fatal") {
				internalOnly = true
			}
		case *ast.AssignStmt:
			for _, lhs := range value.Lhs {
				name := selectorName(lhs)
				if name == "msg" || name == "banner" || name == "endingNotice" {
					directUI = true
				}
			}
		case *ast.KeyValueExpr:
			if key, ok := value.Key.(*ast.Ident); ok && (key.Name == "Label" || key.Name == "Text" || key.Name == "Prompt") {
				directUI = true
			}
		}
	}
	if directUI {
		return "runtime_ui", "direct_ui_context", true
	}
	if semanticRole != "" {
		return semanticRole, "function_context", true
	}
	if internalOnly {
		return "", "", false
	}
	return "go_review", "non_ascii_production_literal", false
}

func collectGo(repo string) ([]Entry, error) {
	root := filepath.Join(repo, "remake", "cmd", "fd2")
	set := token.NewFileSet()
	packages, err := parser.ParseDir(set, root, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, pkg := range packages {
		for filename, file := range pkg.Files {
			var stack []ast.Node
			ast.Inspect(file, func(node ast.Node) bool {
				if node == nil {
					stack = stack[:len(stack)-1]
					return true
				}
				stack = append(stack, node)
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil || text == "" {
					return true
				}
				role, confidence, direct := goRole(stack[:len(stack)-1])
				if !direct && (!hasNonASCII(text) || role == "") {
					return true
				}
				position := set.Position(literal.Pos())
				function := ""
				for index := len(stack) - 2; index >= 0; index-- {
					if declaration, ok := stack[index].(*ast.FuncDecl); ok {
						function = declaration.Name.Name
						break
					}
				}
				relative, err := filepath.Rel(repo, filename)
				if err != nil {
					return true
				}
				id := provisionalID("legacy.go", relative, fmt.Sprintf("l%d-c%d", position.Line, position.Column))
				entries = append(entries, Entry{
					StringID: id, IDStatus: "provisional_source_location", Role: role,
					Confidence: confidence, Text: text, Variables: variables(text),
					Source: Source{File: filepath.ToSlash(relative), Function: function, Line: position.Line, Column: position.Column},
				})
				return true
			})
		}
	}
	return entries, nil
}

func build(repo string) (Inventory, error) {
	jsonEntries, err := collectJSON(repo)
	if err != nil {
		return Inventory{}, err
	}
	goEntries, err := collectGo(repo)
	if err != nil {
		return Inventory{}, err
	}
	entries := append(jsonEntries, goEntries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].StringID < entries[j].StringID })
	seen := make(map[string]Source, len(entries))
	for _, entry := range entries {
		if previous, exists := seen[entry.StringID]; exists {
			return Inventory{}, fmt.Errorf("duplicate string_id %s at %v and %v", entry.StringID, previous, entry.Source)
		}
		seen[entry.StringID] = entry.Source
	}
	return Inventory{
		SchemaVersion: 1, Kind: "fd2_string_inventory", Locale: "zh-Hant", Status: "diagnostic",
		GeneratedBy: map[string]string{"tool": "fd2-string-inventory", "version": "1"},
		Entries:     entries,
	}, nil
}

func main() {
	repo := flag.String("repo", "..", "FD2 repository root")
	output := flag.String("output", "", "output JSON (empty writes stdout)")
	summary := flag.Bool("summary", false, "write compact diagnostic summary instead of the full inventory")
	flag.Parse()
	inventory, err := build(filepath.Clean(*repo))
	if err != nil {
		fmt.Fprintln(os.Stderr, "錯誤：", err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "錯誤：", err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if *summary {
		digest := sha256.Sum256(raw)
		result := Summary{
			SchemaVersion: 1, Kind: "fd2_string_inventory_summary", Locale: inventory.Locale,
			Status: inventory.Status, GeneratedBy: inventory.GeneratedBy,
			InventorySHA256: fmt.Sprintf("%x", digest), EntryCount: len(inventory.Entries),
			ByIDStatus: map[string]int{}, ByRole: map[string]int{}, ByRoleUniqueText: map[string]int{}, ByConfidence: map[string]int{},
		}
		uniqueText := map[string]struct{}{}
		roleText := map[string]map[string]struct{}{}
		for _, entry := range inventory.Entries {
			result.ByIDStatus[entry.IDStatus]++
			result.ByRole[entry.Role]++
			result.ByConfidence[entry.Confidence]++
			uniqueText[entry.Text] = struct{}{}
			if roleText[entry.Role] == nil {
				roleText[entry.Role] = map[string]struct{}{}
			}
			roleText[entry.Role][entry.Text] = struct{}{}
			if len(entry.Variables) > 0 {
				result.VariableEntryCount++
			}
		}
		result.UniqueTextCount = len(uniqueText)
		for role, texts := range roleText {
			result.ByRoleUniqueText[role] = len(texts)
		}
		raw, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "錯誤：", err)
			os.Exit(1)
		}
		raw = append(raw, '\n')
	}
	if *output == "" {
		_, _ = os.Stdout.Write(raw)
		return
	}
	if err := os.WriteFile(*output, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "錯誤：", err)
		os.Exit(1)
	}
}
