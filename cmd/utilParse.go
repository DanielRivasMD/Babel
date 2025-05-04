package cmd

import (
	"fmt"
	"os"
	"strings"

	"olympos.io/encoding/edn"
)

// Package-level variable for the TC prefix.
var TC = "TC"

// ednFile is assumed to be declared externally.

const (
	DefaultKey = " - "
	OutputDir  = "layouts"
	OutputFile = "keyboard_layout.md"
)

// mappingLabels converts an EDN rule key (without any prefix markers) to a friendly label.
var mappingLabels = map[string]string{
	// TODO: add other characters
	"delete_or_backspace": "BACK",
	"return_or_enter":     "ENTER",
	"right_shift":         "SHIFT",
	"right_option":        "ALT",
	"right_command":       "CMD",
	"spacebar":            "SPACE",
}

type KeyboardConfig struct {
	Letters      map[string]string
	Numbers      map[string]string
	SpecialKeys  map[string]string
	UsedTcPrefix string
}

func parse() {
	// ednFile is assumed declared externally.
	config := parseEdnConfig(ednFile)
	// Dump configuration maps for debugging.
	fmt.Println("[DEBUG] Letters:", config.Letters)
	fmt.Println("[DEBUG] Numbers:", config.Numbers)
	fmt.Println("[DEBUG] SpecialKeys:", config.SpecialKeys)

	generateMarkdown(config)
	fmt.Printf("Generated layout using TC variable: '%s'\n", TC)
	fmt.Printf("Output: %s/%s\n", OutputDir, OutputFile)
}

func parseEdnConfig(filePath string) KeyboardConfig {
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error reading EDN file: %v", err))
	}
	var raw interface{}
	if err := edn.Unmarshal(file, &raw); err != nil {
		panic(fmt.Sprintf("Error parsing EDN: %v", err))
	}
	fmt.Printf("[DEBUG] Type of raw: %T\n", raw)
	fmt.Println("[DEBUG] Raw:", raw)

	// Merge multiple EDN documents if needed.
	var docs []map[edn.Keyword]interface{}
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			fmt.Println("[DEBUG] Slice item:", item)
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
		fmt.Printf("[DEBUG] Raw (edn.Keyword keys): %#v\n", docs)
	case map[string]interface{}:
		convMap := make(map[edn.Keyword]interface{})
		for key, value := range v {
			convMap[edn.Keyword(key)] = value
		}
		docs = append(docs, convMap)
		fmt.Printf("[DEBUG] Raw (string keys): %#v\n", docs)
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
		fmt.Printf("[DEBUG] Raw (interface{} keys): %#v\n", docs)
	default:
		fmt.Println("nothing matching")
	}

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
		"open_bracket", "close_bracket", "semicolon", "quote",
		"backslash", "comma", "period", "slash",
		"delete_or_backspace", "return_or_enter",
		"right_shift", "right_option", "right_command", "spacebar",
		"left_arrow", "right_arrow", "up_arrow", "down_arrow",
	}
	for _, key := range specialKeys {
		config.SpecialKeys[key] = DefaultKey
	}
	// Set default overrides.
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
					config.Letters[letter] = val
					fmt.Printf("[DEBUG] Found letter mapping: %s -> %s\n", letter, val)
					break
				}
			}

			// Process number keys.
			for _, d := range digitKeys {
				expected := fmt.Sprintf("!%s#P%s", TC, d)
				if keyStr == expected {
					val := formatEdnValue(value)
					config.Numbers[d] = val
					fmt.Printf("[DEBUG] Found number mapping: %s -> %s\n", d, val)
					break
				}
			}
			// Process dash key.
			if keyStr == fmt.Sprintf("!%s#P-_", TC) || keyStr == fmt.Sprintf(":!%s#P-", TC) {
				val := formatEdnValue(value)
				config.Numbers["-"] = val
				fmt.Printf("[DEBUG] Found number mapping for dash -> %s\n", val)
			}
			// Process equals key.
			if keyStr == fmt.Sprintf("!%s#P+=", TC) || keyStr == fmt.Sprintf(":!%s#P=", TC) {
				val := formatEdnValue(value)
				config.Numbers["="] = val
				fmt.Printf("[DEBUG] Found number mapping for equals -> %s\n", val)
			}

			// Process special keys (using substring matching).
			if strings.Contains(keyStr, "open_bracket") {
				val := formatEdnValue(value)
				config.SpecialKeys["open_bracket"] = val
				fmt.Printf("[DEBUG] Found mapping for open_bracket -> %s\n", val)
			} else if strings.Contains(keyStr, "close_bracket") {
				val := formatEdnValue(value)
				config.SpecialKeys["close_bracket"] = val
				fmt.Printf("[DEBUG] Found mapping for close_bracket -> %s\n", val)
			} else if strings.Contains(keyStr, "delete_or_backspace") {
				val := formatEdnValue(value)
				config.SpecialKeys["delete_or_backspace"] = val
				fmt.Printf("[DEBUG] Found mapping for delete_or_backspace -> %s\n", val)
			} else if strings.Contains(keyStr, "return_or_enter") {
				val := formatEdnValue(value)
				config.SpecialKeys["return_or_enter"] = val
				fmt.Printf("[DEBUG] Found mapping for return_or_enter -> %s\n", val)
			} else if strings.Contains(keyStr, "right_shift") {
				val := formatEdnValue(value)
				config.SpecialKeys["right_shift"] = val
				fmt.Printf("[DEBUG] Found mapping for right_shift -> %s\n", val)
			} else if strings.Contains(keyStr, "right_option") {
				val := formatEdnValue(value)
				config.SpecialKeys["right_option"] = val
				fmt.Printf("[DEBUG] Found mapping for right_option -> %s\n", val)
			} else if strings.Contains(keyStr, "right_command") {
				val := formatEdnValue(value)
				config.SpecialKeys["right_command"] = val
				fmt.Printf("[DEBUG] Found mapping for right_command -> %s\n", val)
			} else if strings.Contains(keyStr, "spacebar") {
				val := formatEdnValue(value)
				config.SpecialKeys["spacebar"] = val
				fmt.Printf("[DEBUG] Found mapping for spacebar -> %s\n", val)
			} else if strings.Contains(keyStr, "left_arrow") {
				val := formatEdnValue(value)
				config.SpecialKeys["left_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for left_arrow -> %s\n", val)
			} else if strings.Contains(keyStr, "right_arrow") {
				val := formatEdnValue(value)
				config.SpecialKeys["right_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for right_arrow -> %s\n", val)
			} else if strings.Contains(keyStr, "up_arrow") {
				val := formatEdnValue(value)
				config.SpecialKeys["up_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for up_arrow -> %s\n", val)
			} else if strings.Contains(keyStr, "down_arrow") {
				val := formatEdnValue(value)
				config.SpecialKeys["down_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for down_arrow -> %s\n", val)
			}
		}
	}
	config.UsedTcPrefix = TC
	return config
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
		config.Numbers["0"], config.Numbers["-"], config.Numbers["="],
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
