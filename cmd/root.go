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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var rootCmd = &cobra.Command{
	Use:     "babel",
	Long:    helpRoot,
	Example: exampleRoot,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func Execute() {
	horus.CheckErr(rootCmd.Execute())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var rg = map[string]*regexp.Regexp{
	"fn": regexp.MustCompile(`^([OESRTWCQ]+)(f[0-9]+)$`),
	"ch": regexp.MustCompile(`^([OESRTWCQ]+)([a-z])$`),
}

// // replace arrows & modifiers
// var triggerFormat = strings.NewReplacer(
// 	"page_up", "pgup",
// 	"page_down", "pgdw",

// 	"up_arrow", "↑",
// 	"down_arrow", "↓",
// 	"right_arrow", "→",
// 	"left_arrow", "←",

// 	"left_shift", "<S>",
// 	"left_control", "<T>",
// 	"left_option", "<O>",
// 	"left_command", "<C>",

// 	"right_shift", "<R>",
// 	"right_control", "<W>",
// 	"right_option", "<E>",
// 	"right_command", "<Q>",

// 	"tab", "TAB",
// 	"delete_or_backspace", "DEL",
// 	"return_or_enter", "RET",
// 	"caps_lock", "<P>",
// 	"spacebar", "<_>",

// 	"hyphen", "-",
// 	"equal_sign", "=",
// 	"open_bracket", "[",
// 	"close_bracket", "]",
// 	"semicolon", ";",
// 	"quote", "'",
// 	"backslash", "\\",
// 	"comma", ",",
// 	"period", ".",
// 	"slash", "/",

// 	"non_us_pound", "•",
// )

var (
	dirs  configDirs
	flags babelFlags
)

type configDirs struct {
	home string
}

type babelFlags struct {
	verbose    bool
	rootDir    string
	program    string
	ednFile    string
	renderMode string
	sortBy     string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose diagnostics")
	rootCmd.PersistentFlags().StringVarP(&flags.program, "program", "", "", "Regex or substring to filter Program names (e.g. helix)")
	rootCmd.PersistentFlags().StringVarP(&flags.rootDir, "root", "", defaultRootDir(), "Config root (recurses .edn files)")

	horus.CheckErr(
		displayCmd.RegisterFlagCompletionFunc("program", completePrograms),
		horus.WithOp("root.init"),
		horus.WithMessage("registering config completion for flag program"),
	)

	cobra.OnInitialize(initConfigDirs)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func initConfigDirs() {
	var err error
	dirs.home, err = domovoi.FindHome(flags.verbose)
	horus.CheckErr(err, horus.WithCategory("init_error"), horus.WithMessage("getting home directory"))
}

func onelineErr(er string) string {
	return chalk.Bold.TextStyle(chalk.Red.Color(er))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add config for binding interpret & display
type KeySeq struct {
	Mode     string
	Modifier string
	Key      string
}

func (s KeySeq) Render() string {
	return fmt.Sprintf("mode:%s - modifier:%s - key:%s", s.Mode, s.Modifier, s.Key)
}

type ProgramAction struct {
	Program string
	Action  string
	Command string
}

func (a ProgramAction) Render() string {
	return fmt.Sprintf("Program: %s | Action: %s | Command: %s", a.Program, a.Action, a.Command)
}

type BindingEntry struct {
	Trigger  KeySeq
	Binding  KeySeq
	Sequence string
	Actions  []ProgramAction
}

func (b BindingEntry) Render() string {
	var actions []string
	for _, a := range b.Actions {
		actions = append(actions, a.Render())
	}
	return fmt.Sprintf(
		"Trigger: [%s] | Binding: [%s] | Sequence: %s\n  Actions:\n    %s",
		b.Trigger.Render(), b.Binding.Render(), b.Sequence,
		strings.Join(actions, "\n    "),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func completeRenderType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"empty", "full", "default"}, cobra.ShellCompDirectiveNoFileComp
}

func completePrograms(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"helix", "helix-common", "helix-insert", "helix-normal", "helix-select", "micro"}, cobra.ShellCompDirectiveNoFileComp
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parseBinding(rawMeta map[edn.Keyword]any, vec []any, mode string) *BindingEntry {
	if len(vec) < 2 {
		return nil // malformed rule vector
	}

	// Parse trigger
	rawTrigger := string(vec[0].(edn.Keyword))
	tm, tk := splitEDNKey(rawTrigger)
	trigger := KeySeq{Mode: mode, Modifier: tm, Key: tk}

	// Parse binding
	rawBinding := buildKeySequence(vec[1])
	bm, bk := splitEDNKey(rawBinding)
	binding := KeySeq{Mode: "", Modifier: bm, Key: bk}

	// Parse actions
	acts, ok := rawMeta[edn.Keyword("doc/actions")].([]any)
	if !ok {
		return nil
	}

	var actions []ProgramAction
	for _, a := range acts {
		m, ok := a.(map[any]any)
		if !ok {
			continue
		}
		actions = append(actions, ProgramAction{
			Program: fmt.Sprint(m[edn.Keyword("program")]),
			Action:  fmt.Sprint(m[edn.Keyword("name")]),
			Command: fmt.Sprint(m[edn.Keyword("exec")]),
		})
	}

	// Optional sequence
	seq := ""
	if v, ok := rawMeta[edn.Keyword("sequence")]; ok {
		seq = fmt.Sprint(v)
	}

	return &BindingEntry{
		Trigger:  trigger,
		Binding:  binding,
		Sequence: seq,
		Actions:  actions,
	}
}

func parseBindings(text, mode string) []BindingEntry {
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

		if entry := parseBinding(rawMeta, vec, mode); entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

func parseEDNFile(path string) ([]BindingEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	text := string(data)
	mode := extractMode(text)
	return parseBindings(text, mode), nil
}

// stripEDNPrefix trims whitespace and any leading EDN prefix ":!"
func stripEDNPrefix(raw string) string {
	s := strings.TrimSpace(raw)
	return strings.TrimPrefix(s, ":!")
}

// splitEDNKey rewrites a Keyword like ":!Tpage_up" → "T page_up"
func splitEDNKey(str string) (string, string) {
	str = strings.TrimPrefix(str, ":")
	str = strings.TrimPrefix(str, "!")
	// if str == "" {
	// 	return "", ""
	// }
	parts := strings.SplitN(str, "#P", 2) // group#Pname
	modifier := parts[0]
	key := ""
	if len(parts) > 1 {
		key = parts[1]
	}
	return modifier, key
}

func gatherRowsFromPaths(paths []string) ([]BindingEntry, error) {
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

// TODO: validate flags on prerun

// filterByProgram applies the optional programFilter regex to a slice of Rows.
// If programFilter is empty, it returns rows unmodified.
func filterByProgram(entries []BindingEntry, programFilter string) []BindingEntry {
	if programFilter == "" {
		return entries
	}
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

// TODO: update default root dir definition
func defaultRootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // will be caught later
	}
	return filepath.Join(home, ".saiyajin", "frag")
}

// TODO: update error habdling
// resolveEDNFiles returns either the single --file or all .edn under --root
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

// extractEntry finds the next ^{…}[…] pair, returns meta & vector & new position
func extractEntry(text string, startPos int) (metaStr, vecStr string, nextPos int, ok bool) {
	// find next caret
	delta := strings.IndexRune(text[startPos:], '^')
	if delta < 0 {
		return "", "", 0, false
	}
	i := startPos + delta

	// skip whitespace, expect '{'
	j := i + 1
	for j < len(text) && unicode.IsSpace(rune(text[j])) {
		j++
	}
	if j >= len(text) || text[j] != '{' {
		return extractEntry(text, i+1)
	}

	// extract metadata map literal
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
				k++ // include closing
				break metaLoop
			}
		}
	}
	if braceCount != 0 {
		return "", "", 0, false
	}
	metaEnd := k
	metaStr = text[j:metaEnd]

	// skip to '['
	p := metaEnd
	for p < len(text) && unicode.IsSpace(rune(text[p])) {
		p++
	}
	if p >= len(text) || text[p] != '[' {
		return extractEntry(text, metaEnd)
	}

	// extract the vector literal
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
				q++ // include closing
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

// loadEDNFile reads the entire EDN file into a string
func loadEDNFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		horus.CheckErr(
			err,
			horus.WithMessage(path),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string {
				return "failed to read: " + chalk.Red.Color(he.Message)
			}),
		)
	}
	return string(data)
}

// decodeMetadata turns the EDN map string into map
func decodeMetadata(metaStr string) (map[edn.Keyword]any, error) {
	var rawMeta map[edn.Keyword]any
	err := edn.Unmarshal([]byte(metaStr), &rawMeta)
	return rawMeta, err
}

// decodeRule parses the EDN vector into []any
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
