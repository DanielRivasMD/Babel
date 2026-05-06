/*
Copyright © 2026 Daniel Rivas <danielrivasmd@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/DanielRivasMD/horus"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func resolveEDNFiles(file, root string) []string {
	if file != "" {
		return []string{file}
	}
	var ednFiles []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".edn") {
			ednFiles = append(ednFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to scan %s: %v", root, err)
	}
	return ednFiles
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func loadEDNFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		horus.CheckErr(err, horus.WithMessage(path), horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string {
				return "failed to read: " + chalk.Red.Color(he.Message)
			}))
	}
	return string(data)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func decodeMetadata(metaStr string) (map[edn.Keyword]any, error) {
	var rawMeta map[edn.Keyword]any
	err := edn.Unmarshal([]byte(metaStr), &rawMeta)
	return rawMeta, err
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func decodeRule(vecStr string) ([]any, error) {
	var raw any
	dec := edn.NewDecoder(strings.NewReader(vecStr))
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	vec, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid rule form")
	}
	return vec, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parseBindingEntry(rawMeta map[edn.Keyword]any, vec []any, mode string) *BindingEntry {
	if len(vec) < 2 {
		return nil
	}
	rawTrigger := string(vec[0].(edn.Keyword))
	tm, tk := splitEDNKey(rawTrigger)
	trigger := KeySeq{Mode: mode, Modifier: tm, Key: tk}

	rawBinding := buildKeySequence(vec[1])
	bm, bk := splitEDNKey(rawBinding)
	binding := KeySeq{Mode: "", Modifier: bm, Key: bk}

	var actions []ProgramAction
	var seq string
	if acts, ok := rawMeta[edn.Keyword("doc/actions")].([]any); ok {
		for _, a := range acts {
			m, ok := a.(map[any]any)
			if !ok {
				continue
			}
			actions = append(actions, ProgramAction{
				Program: fmt.Sprint(m[edn.Keyword("program")]),
				Action:  fmt.Sprint(m[edn.Keyword("action")]),
				Command: fmt.Sprint(m[edn.Keyword("exec")]),
			})
			if raw, ok := m[edn.Keyword("sequence")]; ok && raw != nil {
				seq = fmt.Sprint(raw)
			}
		}
	}

	annotations := parseAnnotations(vec)

	if aloneVals, ok := annotations["alone"]; ok && len(aloneVals) > 0 {
		bm, bk = splitEDNKey(aloneVals[0])
		binding = KeySeq{Mode: "", Modifier: bm, Key: bk}
	}

	return &BindingEntry{
		Trigger:     trigger,
		Binding:     binding,
		Sequence:    seq,
		Actions:     actions,
		Annotations: annotations,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parseBindingEntries(text, mode string) []BindingEntry {
	var entries []BindingEntry
	pos := 0
	for {
		metaStr, vecStr, nextPos, ok := extractEntry(text, pos)
		if !ok {
			break
		}
		pos = nextPos
		rawMeta, err := decodeMetadata(metaStr)
		if err != nil {
			log.Fatalf("EDN metadata unmarshal error: %v", err)
		}
		vec, err := decodeRule(vecStr)
		if err != nil {
			log.Fatalf("EDN rule decode error: %v", err)
		}
		if entry := parseBindingEntry(rawMeta, vec, mode); entry != nil {
			entries = append(entries, *entry)
		}
	}
	return entries
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parseEDNFile(path string) ([]BindingEntry, error) {
	text := loadEDNFile(path)
	mode := extractMode(text)
	return parseBindingEntries(text, mode), nil
}

func parseEDNFiles(paths []string) ([]BindingEntry, error) {
	var all []BindingEntry
	for _, path := range paths {
		entries, err := parseEDNFile(path)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	return all, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func extractEntry(text string, startPos int) (metaStr, vecStr string, nextPos int, ok bool) {
	delta := strings.IndexRune(text[startPos:], '^')
	if delta < 0 {
		return "", "", 0, false
	}
	i := startPos + delta
	j := i + 1
	for j < len(text) && unicode.IsSpace(rune(text[j])) {
		j++
	}
	if j >= len(text) || text[j] != '{' {
		return extractEntry(text, i+1)
	}
	braceCount := 0
	k := j
metaLoop:
	for ; k < len(text); k++ {
		switch text[k] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				k++
				break metaLoop
			}
		}
	}
	if braceCount != 0 {
		return "", "", 0, false
	}
	metaEnd := k
	metaStr = text[j:metaEnd]

	p := metaEnd
	for p < len(text) && unicode.IsSpace(rune(text[p])) {
		p++
	}
	if p >= len(text) || text[p] != '[' {
		return extractEntry(text, metaEnd)
	}
	bracketCount := 0
	q := p
vecLoop:
	for ; q < len(text); q++ {
		switch text[q] {
		case '[':
			bracketCount++
		case ']':
			bracketCount--
			if bracketCount == 0 {
				q++
				break vecLoop
			}
		}
	}
	if bracketCount != 0 {
		return "", "", 0, false
	}
	vecEnd := q
	vecStr = text[p:vecEnd]
	return metaStr, vecStr, vecEnd, true
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func extractMode(text string) string {
	ixSpace := 20
	ruleStart := strings.Index(text, ":rules")
	if ruleStart < 0 {
		return ""
	}
	sliceRule := text[ruleStart : ruleStart+ixSpace]
	brOpen := strings.Index(sliceRule, "[")
	if brOpen < 0 {
		return ""
	}
	if sliceRule[brOpen+1:brOpen+2] != ":" {
		return ""
	}
	sliceMode := sliceRule[brOpen:]
	startMode := strings.Index(sliceMode, ":")
	endMode := strings.Index(sliceMode, "-")
	if startMode < 0 || endMode < 0 {
		return ""
	}
	mode := sliceRule[brOpen:][startMode:endMode]
	mode = strings.TrimPrefix(mode, ":")
	return mode
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parseAnnotations(vec []any) map[string][]string {
	anns := make(map[string][]string)
	if len(vec) < 4 {
		return anns
	}
	annMap, ok := vec[3].(map[any]any)
	if !ok {
		return anns
	}
	for k, v := range annMap {
		kw, ok := k.(edn.Keyword)
		if !ok {
			continue
		}
		key := string(kw)
		switch vv := v.(type) {
		case []any:
			for _, item := range vv {
				anns[key] = append(anns[key], fmt.Sprint(item))
			}
		default:
			anns[key] = append(anns[key], fmt.Sprint(v))
		}
	}
	return anns
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func filterByProgram(entries []BindingEntry, programFilter string) []BindingEntry {
	if programFilter == "" {
		return entries
	}
	progRE, _ := regexp.Compile(programFilter)

	var out []BindingEntry
	for _, e := range entries {
		var filtered []ProgramAction
		for _, a := range e.Actions {
			if progRE.MatchString(a.Program) {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) > 0 {
			e.Actions = filtered
			out = append(out, e)
		}
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func stripEDNPrefix(str string) string {
	str = strings.TrimSpace(str)
	str = strings.TrimPrefix(str, ":")
	str = strings.TrimPrefix(str, "!")
	return str
}

func splitEDNKey(str string) (string, string) {
	str = stripEDNPrefix(str)
	for _, re := range rg {
		if m := re.FindStringSubmatch(str); len(m) == 3 {
			return m[1], m[2]
		}
	}
	return "", str
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func buildKeySequence(x any) string {
	switch kv := x.(type) {
	case []any:
		parts := make([]string, len(kv))
		for i, e := range kv {
			parts[i] = fmt.Sprint(e)
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprint(kv)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatKeySeq(k KeySeq, lookups map[string]KeyLookup, program, sep string) string {
	lookup := lookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = lookups["default"]
	}
	var modParts []string
	for _, r := range k.Modifier {
		modParts = append(modParts, lookup(string(r)))
	}
	mod := strings.Join(modParts, sep)
	key := functionKey2UpperCase(k.Key)
	mapped := lookup(key)
	var out string
	if mod != "" {
		out = mod + sep + mapped
	} else {
		out = mapped
	}
	if k.Mode != "" {
		return "(" + k.Mode + ") " + out
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatTriggerEntry(k KeySeq, lookups map[string]KeyLookup, program string, transforms map[string]string) string {
	lookup := lookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = lookups["default"]
	}
	var modParts []string
	for _, r := range k.Modifier {
		modParts = append(modParts, lookup(string(r)))
	}
	mod := strings.Join(modParts, "") // no separator
	// key := functionKey2UpperCase(k.Key)

	key := transforms[k.Key]

	// mapped := lookup(key)
	var out string
	if mod != "" {
		out = mod + key
	} else {
		out = key
	}
	if k.Mode != "" {
		out = k.Mode + out
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatBindingEntry(b BindingEntry, lookups map[string]KeyLookup, program string) string {
	lookup := lookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = lookups["default"]
	}
	key := b.Sequence
	if key == "" {
		key = b.Binding.Key
	}
	key = functionKey2UpperCase(key)
	var modParts []string
	for _, r := range b.Binding.Modifier {
		modParts = append(modParts, lookup(string(r)))
	}
	mod := strings.Join(modParts, "-")
	var out string
	if mod != "" {
		out = mod + "-" + lookup(key)
	} else {
		out = lookup(key)
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatKanataReplace(trigger, bind string) moldReplace {
	return replace(fmt.Sprintf("  %s  %s:line", trigger, bind), bind)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatZellijReplace(key string, act ProgramAction) moldReplace {
	escapedCmd := escapeForMold(act.Command)
	escapedCmd = strings.Trim(escapedCmd, "[]")
	lhs := escapedCmd
	rhs := fmt.Sprintf("        bind \\\"%s\\\" { %s }:line", key, escapedCmd)
	return replace(fmt.Sprintf("\"%s\"", lhs), rhs)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func escapeForMold(cmd string) string {
	return strings.ReplaceAll(cmd, `"`, `\"`)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: expand normalize to other programs beyond zellij, micro & helix
func normalizeProgram(p string) string {
	if p == "kanata" {
		return "kanata"
	}
	switch {
	case strings.Contains(p, "zellij"):
		return "zellij"
	case strings.Contains(p, "micro"):
		return "micro"
	case strings.Contains(p, "helix"):
		return "helix"
	default:
		return p
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func formatBinds(raw map[string]string, program string) map[string]string {
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		var prettyVal string
		switch {
		case strings.HasPrefix(program, "helix-"):
			prettyVal = tomlList(v)
		case program == "micro",
			program == "lazygit",
			program == "serpl",
			program == "zellij":
			prettyVal = strings.Trim(v, "[]")
		default:
			prettyVal = v
		}
		out[k] = prettyVal
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func tomlList(raw string) string {
	inner := strings.TrimSpace(raw)
	inner = strings.TrimPrefix(inner, "[")
	inner = strings.TrimSuffix(inner, "]")
	if strings.HasPrefix(inner, ":sh ") || strings.HasPrefix(inner, ":echo ") {
		return fmt.Sprintf("[%q]", inner)
	}
	if inner == "" {
		return "[]"
	}
	parts := strings.Fields(inner)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func functionKey2UpperCase(key string) string {
	if strings.HasPrefix(strings.ToLower(key), "f") && len(key) > 1 {
		if _, err := strconv.Atoi(key[1:]); err == nil {
			return "F" + key[1:]
		}
	}
	return key
}

////////////////////////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////////////////////////
