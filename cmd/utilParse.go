////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"os"
	"strings"

	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

// Package-level variable for the TC prefix.
var TC = "TC"

// ednFile is assumed to be declared externally.

const (
	DefaultKey = " "
	OutputDir  = "layouts"
	OutputFile = "keyboard_layout.md"
)

// mappingLabels converts an EDN rule key (without any prefix markers) to a friendly label.
var mappingLabels = map[string]string{
	"hyphen":              "-",
	"equal_sign":          "=",
	"delete_or_backspace": "BACK",
	"return_or_enter":     "ENTER",
	"right_shift":         "SHIFT",
	"right_option":        "ALT",
	"right_command":       "CMD",
	"spacebar":            "SPACE",
	// TODO: add lefts
}

type KeyboardConfig struct {
	Letters      map[string]string
	Numbers      map[string]string
	SpecialKeys  map[string]string
	UsedTcPrefix string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parse() {
	// ednFile is assumed declared externally.
	config := parseEdnConfig(ednFile)
	// Dump configuration maps for debugging.
	// fmt.Println("[DEBUG] Letters:", config.Letters)
	// fmt.Println("[DEBUG] Numbers:", config.Numbers)
	// fmt.Println("[DEBUG] SpecialKeys:", config.SpecialKeys)

	generateMarkdown(config)

	if verbose {
		fmt.Printf("Generated layout using TC variable: '%s'\n", TC)
		fmt.Printf("Output: %s/%s\n", OutputDir, OutputFile)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func parseEdnConfig(filePath string) KeyboardConfig {
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error reading EDN file: %v", err))
	}
	var raw interface{}
	if err := edn.Unmarshal(file, &raw); err != nil {
		panic(fmt.Sprintf("Error parsing EDN: %v", err))
	}

	// Merge multiple EDN documents if needed.
	var docs []map[edn.Keyword]interface{}
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			// fmt.Println("[DEBUG] Slice item:", item)
			if m, ok := item.(map[edn.Keyword]interface{}); ok {
				docs = append(docs, m)
			} else if m, ok := item.(map[interface{}]interface{}); ok {
				convMap := make(map[edn.Keyword]interface{})
				for key, value := range m {
					var k edn.Keyword
					switch t := key.(type) {
					case string:
						k = edn.Keyword(t)
					default:
						k = edn.Keyword(fmt.Sprintf("%v", t))
					}
					convMap[k] = value
				}
				docs = append(docs, convMap)
			}
		}
	case map[edn.Keyword]interface{}:
		docs = append(docs, v)
		// fmt.Printf("[DEBUG] Raw (edn.Keyword keys): %#v\n", docs)
	case map[string]interface{}:
		convMap := make(map[edn.Keyword]interface{})
		for key, value := range v {
			convMap[edn.Keyword(key)] = value
		}
		docs = append(docs, convMap)
		// fmt.Printf("[DEBUG] Raw (string keys): %#v\n", docs)
	case map[interface{}]interface{}:
		convMap := make(map[edn.Keyword]interface{})
		for key, value := range v {
			var k edn.Keyword
			switch t := key.(type) {
			case string:
				k = edn.Keyword(t)
			default:
				k = edn.Keyword(fmt.Sprintf("%v", t))
			}
			convMap[k] = value
		}
		docs = append(docs, convMap)
		// fmt.Printf("[DEBUG] Raw (interface{} keys): %#v\n", docs)
	default:
		// TODO: error out gracefully
		fmt.Println("nothing matching")
	}

	// TODO: wrap initialization elements into function
	// Initialize configuration.
	config := KeyboardConfig{
		Letters:     make(map[string]string),
		Numbers:     make(map[string]string),
		SpecialKeys: make(map[string]string),
	}

	// Initialize letter keys (a-z) with default.
	for c := 'a'; c <= 'z'; c++ {
		config.Letters[string(c)] = DefaultKey
	}

	// Initialize number keys: digits 1-0, dash and equals.
	digitKeys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	for _, d := range digitKeys {
		config.Numbers[d] = DefaultKey
	}
	config.Numbers["-"] = DefaultKey
	config.Numbers["="] = DefaultKey

	// Initialize special keys.
	specialKeys := []string{
		"hyphen", "equal_sign",
		"open_bracket", "close_bracket",
		"semicolon", "quote", "backslash",
		"comma", "period", "slash",
		"delete_or_backspace", "return_or_enter",
		"right_shift", "right_option", "right_command", "spacebar",
		"left_arrow", "right_arrow", "up_arrow", "down_arrow",
	}
	for _, key := range specialKeys {
		config.SpecialKeys[key] = DefaultKey
	}
	// Set default overrides.
	config.SpecialKeys["hyphen"] = DefaultKey
	config.SpecialKeys["equal_sign"] = DefaultKey
	config.SpecialKeys["open_bracket"] = DefaultKey
	config.SpecialKeys["close_bracket"] = DefaultKey
	config.SpecialKeys["semicolon"] = DefaultKey
	config.SpecialKeys["quote"] = DefaultKey
	config.SpecialKeys["backslash"] = DefaultKey
	config.SpecialKeys["comma"] = DefaultKey
	config.SpecialKeys["period"] = DefaultKey
	config.SpecialKeys["slash"] = DefaultKey
	config.SpecialKeys["delete_or_backspace"] = DefaultKey
	config.SpecialKeys["return_or_enter"] = DefaultKey
	config.SpecialKeys["right_shift"] = DefaultKey
	config.SpecialKeys["right_option"] = DefaultKey
	config.SpecialKeys["right_command"] = DefaultKey
	config.SpecialKeys["spacebar"] = DefaultKey

	// Process EDN rules.
	for _, doc := range docs {
		rulesRaw, ok := doc[edn.Keyword(":rules")]
		if !ok {
			continue
		}
		rules, ok := rulesRaw.([]interface{})
		if !ok {
			continue
		}
		for _, rule := range rules {
			ruleList, ok := rule.([]interface{})
			if !ok || len(ruleList) < 2 {
				continue
			}
			key, ok := ruleList[0].(edn.Keyword)
			if !ok {
				continue
			}
			keyStr := string(key)
			value := ruleList[1]

			// Process letter keys (a-z).
			for c := 'a'; c <= 'z'; c++ {
				letter := string(c)
				expected := fmt.Sprintf("!%s#P%s", TC, letter)
				if keyStr == expected {
					val := formatEdnValue(value)
					config.Letters[letter] = chalk.Red.Color(val)
					// fmt.Printf("[DEBUG] Found letter mapping: %s -> %s\n", letter, val)
					break
				}
			}

			// Process number keys.
			for _, d := range digitKeys {
				expected := fmt.Sprintf("!%s#P%s", TC, d)
				if keyStr == expected {
					val := formatEdnValue(value)
					config.Numbers[d] = val
					// fmt.Printf("[DEBUG] Found number mapping: %s -> %s\n", d, val)
					break
				}
			}

			// Build the comment map from the raw EDN file.
			commentMap := buildCommentMap(ednFile)

			// Define the list of target substrings for special keys.
			specialTargets := []string{
				"Phyphen",     // will become "hyphen"
				"Pequal_sign", // will become "equal_sign"
				"Popen_bracket",
				"Pclose_bracket",
				"Psemicolon",
				"Pquote",
				"Pbackslash",
				"Pcomma",
				"Pperiod",
				"Pslash",
				"Pdelete_or_backspace",
				"Preturn_or_enter",
				"Pright_shift",
				"Pright_option",
				"Pright_command",
				"Pspacebar",
				"Pleft_arrow",
				"Pright_arrow",
				"Pup_arrow",
				"Pdown_arrow",
			}

			for _, rule := range rules {
				ruleList, ok := rule.([]interface{})
				if !ok || len(ruleList) < 2 {
					continue
				}
				key, ok := ruleList[0].(edn.Keyword)
				if !ok {
					continue
				}
				keyStr := string(key)
				value := ruleList[1]

				// Process letter keys and number keys separately (as before)...

				// Process special keys:
				for _, target := range specialTargets {
					if processSpecialMapping(&config, commentMap, keyStr, value, target) {
						break // if a special map was processed, no need to continue for this rule.
					}
				}
			}
		}
	}
	config.UsedTcPrefix = TC
	return config
}

func buildCommentMap(filePath string) map[string]bool {
	commentMap := make(map[string]bool)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return commentMap // return an empty map on error
	}
	lines := strings.Split(string(data), "\n")
	// This regex captures the key following "[:!TC#P" until a space, semicolon, or closing bracket.
	for _, line := range lines {
		if strings.Contains(line, "  [:!"+TC+"#P") {
			keys := strings.Split(line, " ")
			if len(keys) >= 2 {
				key := strings.Split(keys[2], "#P")[1]
				fields := strings.Split(line, ";")
				// If there are more than three fields, we consider the rule as having an extra comment.
				hasComment := len(fields) >= 3
				commentMap[key] = hasComment
			}
		}
	}
	return commentMap
}

// derivedKey converts a target like "Phyphen" or "Popen_bracket" into the final config key.
func derivedKey(target string) string {
	if strings.HasPrefix(target, "P") {
		return strings.TrimPrefix(target, "P")
	}
	return target
}

// processSpecialMapping processes a special key if keyStr contains the target substring.
// It uses commentMap (built from the EDN file) to decide if the current rule is commented.
// If commented, it uses bold yellow coloring; otherwise, bold cyan.
// Returns true if a mapping is performed.
func processSpecialMapping(config *KeyboardConfig, commentMap map[string]bool, keyStr string, value interface{}, target string) bool {
	if strings.Contains(keyStr, target) {
		dk := derivedKey(target)
		hasComment := false
		if v, ok := commentMap[dk]; ok {
			hasComment = v
		}
		s := formatEdnValue(value)
		var colored string
		if hasComment {
			// If commented, use bold yellow.
			colored = chalk.Bold.TextStyle(chalk.Yellow.Color(s))
		} else {
			// Otherwise, use bold cyan.
			colored = chalk.Bold.TextStyle(chalk.Cyan.Color(s))
		}
		config.SpecialKeys[dk] = colored
		return true
	}
	return false
}

func formatEdnValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var parts []string
		for _, item := range v {
			switch x := item.(type) {
			case int, int8, int16, int32, int64, float32, float64:
				s := fmt.Sprintf("%v", x)
				fmt.Printf("[DEBUG] Converting numeric value %v -> %s\n", x, s)
				parts = append(parts, s)
			case edn.Keyword:
				str := string(x)
				// Trim any leading ":!" or "!".
				trimmed := strings.TrimPrefix(str, ":!")
				trimmed = strings.TrimPrefix(trimmed, "!")
				parts = append(parts, trimmed)
			case string:
				trimmed := strings.TrimPrefix(x, "!")
				parts = append(parts, trimmed)
			default:
				parts = append(parts, fmt.Sprint(item))
			}
		}
		return strings.Join(parts, " ")
	case edn.Keyword:
		str := string(v)
		trimmed := strings.TrimPrefix(str, ":!")
		trimmed = strings.TrimPrefix(trimmed, "!")
		return trimmed
	case int, int8, int16, int32, int64, float32, float64:
		s := fmt.Sprintf("%v", v)
		fmt.Printf("[DEBUG] Converting numeric value %v -> %s\n", v, s)
		return s
	default:
		return fmt.Sprint(v)
	}
}

// extractMappingComments extracts mapping comments using a regular expression that captures the key name.
func extractMappingComments(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	rawText := string(data)
	lines := strings.Split(rawText, "\n")
	var comments []string
	for _, line := range lines {
		if strings.Contains(line, "[:!TC#P") {
			parts := strings.Split(line, ";")
			if len(parts) >= 3 {
				commentVal := strings.TrimSpace(parts[1])
				commentKey := strings.TrimSpace(parts[2])
				if commentVal != "" && commentKey != "" {
					comments = append(comments, fmt.Sprintf("%s => %s", commentKey, commentVal))
				}
			}
		}
	}
	return comments
}

func generateMarkdown(config KeyboardConfig) {
	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		panic(fmt.Sprintf("Error creating output directory: %v", err))
	}

	file, err := os.Create(fmt.Sprintf("%s/%s", OutputDir, OutputFile))
	if err != nil {
		panic(fmt.Sprintf("Error creating output file: %v", err))
	}
	defer file.Close()

	center := func(text string, width int) string {
		if len(text) >= width {
			return text
		}
		padding := (width - len(text)) / 2
		return fmt.Sprintf("%*s%s%*s", padding, "", text, padding, "")
	}

	markdownStart := fmt.Sprintf(`# Dynamic Keyboard Layout
*Generated with TC='%s'*

`, config.UsedTcPrefix)
	codeFenceStart := "```markdown\n"
	codeFenceEnd := "```\n"

	// Build the dynamic number row. The final cell uses the value for delete_or_backspace.
	numberRow := fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |",
		"~ `",
		config.Numbers["1"], config.Numbers["2"], config.Numbers["3"],
		config.Numbers["4"], config.Numbers["5"], config.Numbers["6"],
		config.Numbers["7"], config.Numbers["8"], config.Numbers["9"],
		config.Numbers["0"], config.SpecialKeys["hyphen"], config.SpecialKeys["equal_sign"],
		config.SpecialKeys["delete_or_backspace"],
	)

	topBorder := "┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬───────────┐\n"
	secondRow := fmt.Sprintf("| TAB | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
		center(config.Letters["q"], 3), center(config.Letters["w"], 3), center(config.Letters["e"], 3),
		center(config.Letters["r"], 3), center(config.Letters["t"], 3), center(config.Letters["y"], 3),
		center(config.Letters["u"], 3), center(config.Letters["i"], 3), center(config.Letters["o"], 3),
		center(config.Letters["p"], 3), config.SpecialKeys["open_bracket"], config.SpecialKeys["close_bracket"],
		center(config.SpecialKeys["backslash"], 8),
	)

	thirdRow := fmt.Sprintf("| CAPS | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |      %s      |\n",
		center(config.Letters["a"], 3), center(config.Letters["s"], 3), center(config.Letters["d"], 3),
		center(config.Letters["f"], 3), center(config.Letters["g"], 3), center(config.Letters["h"], 3),
		center(config.Letters["j"], 3), center(config.Letters["k"], 3), center(config.Letters["l"], 3),
		config.SpecialKeys["semicolon"], config.SpecialKeys["quote"], center(config.SpecialKeys["return_or_enter"], 8),
	)
	fourthRow := fmt.Sprintf("| SHIFT  | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |     %s     |\n",
		center(config.Letters["z"], 3), center(config.Letters["x"], 3), center(config.Letters["c"], 3),
		center(config.Letters["v"], 3), center(config.Letters["b"], 3), center(config.Letters["n"], 3),
		center(config.Letters["m"], 3), config.SpecialKeys["comma"], config.SpecialKeys["period"],
		config.SpecialKeys["slash"], center(config.SpecialKeys["right_shift"], 8),
	)
	fifthRow := fmt.Sprintf("| CTL | ALT | CMD │               %s               │ %s | %s │\n",
		center(config.SpecialKeys["spacebar"], 16),
		config.SpecialKeys["right_command"],
		config.SpecialKeys["right_option"],
	)
	bottomBorder := "└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴───────────┘\n"

	finalLayout := topBorder + numberRow + "\n" + secondRow + thirdRow + fourthRow + fifthRow + bottomBorder

	activeMappingsSection := fmt.Sprintf("\n### Active Mappings\n- **Letters**: %s\n- **Specials**: %s (SPACE), %s (ENTER)\n- **Arrows**: %s\n- **TC Variable**: '%s' (change in script)\n",
		getActiveMappings(config.Letters),
		config.SpecialKeys["spacebar"],
		config.SpecialKeys["return_or_enter"],
		getArrowMappings(config),
		config.UsedTcPrefix,
	)

	mappingComments := extractMappingComments(ednFile)
	mappingCommentsSection := ""
	if len(mappingComments) > 0 {
		mappingCommentsSection = "\n### Mapping Comments\n"
		for _, comment := range mappingComments {
			mappingCommentsSection += "- " + comment + "\n"
		}
	}

	finalContent := markdownStart + codeFenceStart + finalLayout + codeFenceEnd +
		activeMappingsSection + mappingCommentsSection

	if _, writeErr := file.WriteString(finalContent); writeErr != nil {
		panic(fmt.Sprintf("Error writing to output file: %v", writeErr))
	}
}

func getActiveMappings(letters map[string]string) string {
	var active []string
	for c := 'a'; c <= 'z'; c++ {
		letter := string(c)
		if letters[letter] != DefaultKey {
			active = append(active, fmt.Sprintf("%s: %s", letter, letters[letter]))
		}
	}
	if len(active) == 0 {
		return "None"
	}
	return strings.Join(active, ", ")
}

func getArrowMappings(config KeyboardConfig) string {
	arrows := []string{"left_arrow", "right_arrow", "up_arrow", "down_arrow"}
	var mappings []string
	for _, arrow := range arrows {
		if val, ok := config.SpecialKeys[arrow]; ok && val != DefaultKey {
			mappings = append(mappings, fmt.Sprintf("%s: %s", arrow, val))
		}
	}
	if len(mappings) == 0 {
		return "None"
	}
	return strings.Join(mappings, ", ")
}
