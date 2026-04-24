/*
Copyright © 2025 Daniel Rivas <danielrivasmd@gmail.com>

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
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

type KeySeq struct {
	Mode     string
	Modifier string
	Key      string
}

type ProgramAction struct {
	Program string
	Action  string
	Command string
}

type BindingEntry struct {
	Trigger     KeySeq
	Binding     KeySeq
	Sequence    string
	Actions     []ProgramAction
	Annotations map[string][]string
}

type lookUps struct {
	displayBinding map[string]KeyLookup
	displayTrigger map[string]KeyLookup
	interpret      map[string]KeyLookup
	embed          map[string]KeyLookup
}

type KeyLookup func(string) string

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	rg = map[string]*regexp.Regexp{
		"fn": regexp.MustCompile(`^([OESRTWCQ]+)(f[0-9]+)$`),
		"ch": regexp.MustCompile(`^([OESRTWCQ]+)([a-z])$`),
		"nb": regexp.MustCompile(`^([OESRTWCQ]+)([0-9])$`),
		"ot": regexp.MustCompile(`^([OESRTWCQ]+)([a-z_]+)$`),
		"kw": regexp.MustCompile(`^([OESRTWCQ]*)#P(.+)$`),
	}

	lookups lookUps

	programColors = map[string]chalk.Color{
		"micro":        chalk.Cyan,
		"helix-common": chalk.Cyan,
		"helix-insert": chalk.Cyan,
		"helix-normal": chalk.Cyan,
		"helix-pop":    chalk.Cyan,
		"helix-select": chalk.Cyan,
		"broot":        chalk.Green,
		"lazygit":      chalk.Green,
		"serpl":        chalk.Green,
		"terminal":     chalk.Blue,
		"R":            chalk.Blue,
		"zellij":       chalk.Yellow,
	}

	tableBorder  = "=================================================================================================="
	tableHeader  = "| Program         | Action                         | Trigger              | Binding              |"
	tableDivider = "|-----------------|--------------------------------|----------------------|----------------------|"

	TC = "TC" // prefix used in EDN parsing for markdown generation

	// TODO: move away from hardcoded variables => load from template
	programHeaders = map[string][]string{
		"helix-common": {},
		"helix-insert": {
			"[keys.insert]",
			`A-ret = ["completion"]`,
		},
		"helix-normal": {
			"[keys.normal]",
			`A-ret = ["hover"]`,
			// `"`" = "no_op"`,
			// `"A-`" = "no_op"`,
			`g = ["repeat_last_motion"]`,
			`Z = "no_op"`,
			`"~" = "no_op"	`,
			`"=" = "no_op"`,
			`"<" = "no_op"`,
			`">" = "no_op"`,
			`q = "no_op"`,
			`Q = "no_op"`,
			`"|" = "no_op"`,
			`"A-|" = "no_op"`,
			`"!" = "no_op"`,
			`"A-!" = "no_op"`,
			`"$" = "no_op"`,
			`S = "no_op"`,
			`"A-_" = "no_op"`,
			`"&" = "no_op"`,
			`"_" = "no_op"`,
			`"A-;" = "no_op"`,
			`"A-:" = "no_op"`,
			`"," = "no_op"`,
			`"A-," = "no_op"`,
			`C = "no_op"`,
			`"(" = "no_op"`,
			`")" = "no_op"`,
			`"A-(" = "no_op"`,
			`"A-)" = "no_op"`,
			`"%" = "no_op"`,
			`x = "no_op"`,
			`X = "no_op"`,
			`J = "no_op"`,
			`K = "no_op"`,
			`"C-c" = "no_op"`,
			`"/" = "no_op"`,
			`"?" = "no_op"`,
			`m = "no_op"`,
			`n = "no_op"`,
			`N = "no_op"`,
			`"A-*" = "no_op"`,
		},
		"helix-select": {
			"[keys.select]",
			`A-ret = ["hover"]`,
		},
		"micro": {
			`"MouseRight": "MouseMultiCursor",`,
			`"AltEnter": "Autocomplete",`,
		},
	}
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func defaultRootDir() string {
	home, err := domovoi.FindHome(false)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
	return filepath.Join(home, ".saiyajin", "edn")
}

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
	// TODO: replace with horus.OneLineErr
	progRE, err := regexp.Compile(programFilter)
	if err != nil {
		log.Fatalf("invalid --program pattern %q: %v", programFilter, err)
	}
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
	key := normalizeFunctionKey(k.Key)
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

func formatBindingEntry(b BindingEntry, lookups map[string]KeyLookup, program string) string {
	lookup := lookups[normalizeProgram(program)]
	if lookup == nil {
		lookup = lookups["default"]
	}
	key := b.Sequence
	if key == "" {
		key = b.Binding.Key
	}
	key = normalizeFunctionKey(key)
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

func loadFormat(path string) map[string]map[string]string {
	var cfg map[string]map[string]string
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Fatalf("failed to load format config from %s: %v", path, err)
	}
	return cfg
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func buildLookupFuncs(cfg map[string]map[string]string) map[string]KeyLookup {
	defaultMap := cfg["default"]
	out := make(map[string]KeyLookup)
	for program, mapping := range cfg {
		local := mapping
		out[program] = func(local map[string]string) KeyLookup {
			return func(key string) string {
				if val, ok := local[key]; ok {
					return val
				}
				if val, ok := defaultMap[key]; ok {
					return val
				}
				return key
			}
		}(local)
	}
	out["default"] = func(key string) string {
		if val, ok := defaultMap[key]; ok {
			return val
		}
		return key
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func normalizeProgram(p string) string {
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

func normalizeFunctionKey(key string) string {
	if strings.HasPrefix(strings.ToLower(key), "f") && len(key) > 1 {
		if _, err := strconv.Atoi(key[1:]); err == nil {
			return "F" + key[1:]
		}
	}
	return key
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: load completions from toml or const
func completeRenderType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"empty", "full", "default"}, cobra.ShellCompDirectiveNoFileComp
}

func completeSortType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	options := []string{"program", "action", "trigger", "binding"}
	var completions []string
	for _, opt := range options {
		if strings.HasPrefix(opt, toComplete) {
			completions = append(completions, opt)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completePrograms(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"helix", "helix-common", "helix-insert", "helix-normal", "helix-select", "micro", "serpl"}, cobra.ShellCompDirectiveNoFileComp
}

type tableRow struct {
	Program string
	Action  string
	Trigger string
	Binding string
	Empty   bool
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func renderCell(val string, width int, color *chalk.Color) string {
	raw := fmt.Sprintf("%-*s", width, val)
	if color != nil {
		return color.Color(raw)
	}
	return raw
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func isEmptyEntry(e BindingEntry) bool {
	if len(e.Actions) == 0 {
		return true
	}
	for _, a := range e.Actions {
		prog := strings.TrimSpace(fmt.Sprint(a.Program))
		act := strings.TrimSpace(fmt.Sprint(a.Action))
		cmd := strings.TrimSpace(fmt.Sprint(a.Command))
		if prog != "" && prog != "<nil>" {
			return false
		}
		if act != "" && act != "<nil>" {
			return false
		}
		if cmd != "" && cmd != "<nil>" {
			return false
		}
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type moldReplace struct {
	old string
	new string
}

type moldForge struct {
	out      string
	files    []string
	replaces []moldReplace
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func newMoldConfig(outFile string, inFiles []string, replaces ...moldReplace) moldForge {
	return moldForge{
		out:      outFile,
		files:    inFiles,
		replaces: replaces,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func moldForging(op string, mf moldForge) {
	horus.CheckErr(
		domovoi.ExecSh(mf.Cmd()),
		horus.WithOp(op),
		horus.WithCategory("shell_command"),
		horus.WithMessage("Failed to execute mbombo command"),
		horus.WithDetails(map[string]any{
			"command": mf.Cmd(),
		}),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func replace(key, val string) moldReplace {
	return moldReplace{old: key, new: val}
}

func (m moldForge) Cmd() string {
	var files []string
	for _, f := range m.files {
		files = append(files, fmt.Sprintf(`--files %s`, f))
	}
	fileBlock := strings.Join(files, " \\\n")
	var replaces []string
	for _, r := range m.replaces {
		replaces = append(replaces, fmt.Sprintf(`--replace %s="%s"`, r.old, r.new))
	}
	replaceBlock := strings.Join(replaces, " \\\n")
	return fmt.Sprintf(
		`mbombo \
--out %s \
%s \
%s`,
		m.out,
		fileBlock,
		replaceBlock,
	)
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

func printTable(rows []tableRow) {
	if len(rows) == 0 {
		fmt.Println("No bindings found.")
		return
	}
	fmt.Println(tableBorder)
	fmt.Println(tableHeader)
	fmt.Println(tableDivider)
	for _, r := range rows {
		var progColor *chalk.Color
		if c, ok := programColors[r.Program]; ok {
			progColor = &c
		}
		row := fmt.Sprintf("| %s | %s | %s | %s |\n",
			renderCell(r.Program, 15, progColor),
			renderCell(r.Action, 30, nil),
			renderCell(r.Trigger, 20, nil),
			renderCell(r.Binding, 20, nil),
		)
		if r.Empty {
			row = chalk.Dim.TextStyle(row)
		}
		fmt.Print(row)
	}
	fmt.Println(tableBorder)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func embedConfig(entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	switch {
	case target == "kanata":
		replaces := []moldReplace{}
		for _, entry := range entries {
			bindKey := formatKeySeq(entry.Binding, lookups.embed, "kanata", " ")
			triggerKey := formatKeySeq(entry.Trigger, lookups.embed, "kanata", " ")
			replaces = append(replaces, formatKanataReplace(bindKey, triggerKey))
		}
		moldForging(
			"embed-kanata",
			newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...),
		)

	case target == "serpl":
		embedBindings(entries, target, func(key, val string) string {
			return fmt.Sprintf("\\\"<%s>\\\" = \\\"%s\\\":line", key, val)
		})

	case target == "lazygit":
		embedBindings(entries, target, func(key, val string) string {
			return fmt.Sprintf("    %s: '<%s>':line", val, key)
		})

	case strings.HasPrefix(target, "zellij"):
		normalized := normalizeProgram(target)
		replaces := []moldReplace{}
		for _, entry := range filtered {
			for _, act := range entry.Actions {
				bindKey := formatKeySeq(entry.Binding, lookups.embed, normalized, " ")
				replaces = append(replaces, formatZellijReplace(bindKey, act))
			}
		}
		moldForging(
			"embed-zellij",
			newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...),
		)

	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// embedBindings generates replacements for a target program
// formatFunc receives (key, val) and returns the replacement string (including :line suffix if needed)
func embedBindings(entries []BindingEntry, target string, formatFunc func(key, val string) string) {
	filtered := filterByProgram(entries, target)

	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, act := range entry.Actions {
			bindKey := formatKeySeq(entry.Binding, lookups.embed, act.Program, "-")
			rawBind[bindKey] = act.Command
		}
	}
	formatted := formatBinds(rawBind, target)

	replaces := []moldReplace{}
	for key, val := range formatted {
		newLine := formatFunc(key, val)
		replaces = append(replaces, replace(val, newLine))
	}

	mf := newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...)
	moldForging(fmt.Sprintf("embed-%s", target), mf)
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

func emitConfig(w io.Writer, entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)
	rawBind := make(map[string]string)
	for _, entry := range filtered {
		for _, actions := range entry.Actions {
			bindKey := formatKeySeq(entry.Binding, lookups.interpret, actions.Program, "-")
			rawBind[bindKey] = actions.Command
		}
	}
	formatted := formatBinds(rawBind, target)
	switch {
	case strings.HasPrefix(target, "helix-"):
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "%s = %s\n", key, val)
		}
	case target == "micro":
		fmt.Fprintln(w, "{")
		if headerLines, ok := programHeaders[target]; ok {
			for _, line := range headerLines {
				fmt.Fprintln(w, line)
			}
		}
		for key, val := range formatted {
			fmt.Fprintf(w, "  %q: %q,\n", key, val)
		}
		fmt.Fprintln(w, "}")
	default:
		log.Fatalf("unsupported --program %q", target)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
