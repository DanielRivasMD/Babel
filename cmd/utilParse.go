package cmd

import (
	"fmt"
	"os"
	"strings"

	"olympos.io/encoding/edn"
)

// Package-level variable
var TC = "TC"

const (
	DefaultKey = " - "
	OutputDir  = "layouts"
	OutputFile = "keyboard_layout.md"
)

type KeyboardConfig struct {
	Letters      map[string]string
	SpecialKeys  map[string]string
	UsedTcPrefix string
}

func parse() {
	// ednFile is assumed to be declared elsewhere.
	config := parseEdnConfig(ednFile)
	generateMarkdown(config)
	fmt.Printf("Generated layout using TC variable: '%s'\n", TC)
	fmt.Printf("Output: %s/%s\n", OutputDir, OutputFile)
}

func parseEdnConfig(filePath string) KeyboardConfig {
	// Read and unmarshal the EDN file.
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error reading EDN file: %v", err))
	}

	var raw interface{}
	if err := edn.Unmarshal(file, &raw); err != nil {
		panic(fmt.Sprintf("Error parsing EDN: %v", err))
	}
	fmt.Println(raw)
	fmt.Printf("Type of raw: %T\n", raw)

	// Merge multiple EDN documents if needed.
	var docs []map[edn.Keyword]interface{}
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			fmt.Println(item)
			// Try as map[edn.Keyword]interface{}
			if m, ok := item.(map[edn.Keyword]interface{}); ok {
				docs = append(docs, m)
			} else if m, ok := item.(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[edn.Keyword]interface{}
				convMap := make(map[edn.Keyword]interface{})
				for key, value := range m {
					convMap[edn.Keyword(key)] = value
				}
				docs = append(docs, convMap)
			} else if m, ok := item.(map[interface{}]interface{}); ok {
				convMap := make(map[edn.Keyword]interface{})
				for key, value := range m {
					// Try to convert each key to edn.Keyword.
					var k edn.Keyword
					switch t := key.(type) {
					case string:
						k = edn.Keyword(t)
					case edn.Keyword:
						k = t
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
		fmt.Printf("[DEBUG] Raw is a map. Merged docs: %#v\n", docs)
	case map[string]interface{}:
		convMap := make(map[edn.Keyword]interface{})
		for key, value := range v {
			convMap[edn.Keyword(key)] = value
		}
		docs = append(docs, convMap)
		fmt.Printf("[DEBUG] Raw is a map with string keys, converted to edn.Keyword. Merged docs: %#v\n", docs)
	case map[interface{}]interface{}:
		convMap := make(map[edn.Keyword]interface{})
		for key, value := range v {
			var k edn.Keyword
			switch t := key.(type) {
			case string:
				k = edn.Keyword(t)
			case edn.Keyword:
				k = t
			default:
				k = edn.Keyword(fmt.Sprintf("%v", t))
			}
			convMap[k] = value
		}
		docs = append(docs, convMap)
		fmt.Printf("[DEBUG] Raw is a map[interface{}]interface{}, converted: %#v\n", docs)
	default:
		fmt.Println("nothing matching")
	}

	config := KeyboardConfig{
		Letters:     make(map[string]string),
		SpecialKeys: make(map[string]string),
	}

	// Initialize letters a–z.
	for c := 'a'; c <= 'z'; c++ {
		config.Letters[string(c)] = DefaultKey
	}

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
	// Set default override labels.
	config.SpecialKeys["delete_or_backspace"] = "BACK"
	config.SpecialKeys["return_or_enter"] = "ENTER"
	config.SpecialKeys["right_shift"] = "SHIFT"
	config.SpecialKeys["right_option"] = "ALT"
	config.SpecialKeys["right_command"] = "CMD"
	config.SpecialKeys["spacebar"] = "SPACE"

	// Process each document's :rules.
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

			// Process letter keys (e.g. [:!TC#Pa ...]).
			for c := 'a'; c <= 'z'; c++ {
				letter := string(c)
				expected := fmt.Sprintf(":!%s#P%s", TC, letter)
				if keyStr == expected {
					val := formatEdnValue(value)
					config.Letters[letter] = val
					fmt.Printf("[DEBUG] Found letter mapping: %s -> %s\n", letter, val)
					break
				}
			}

			// Process special keys.
			switch keyStr {
			case fmt.Sprintf(":!%s#Popen_bracket", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["open_bracket"] = val
				fmt.Printf("[DEBUG] Found mapping for open_bracket -> %s\n", val)
			case fmt.Sprintf(":!%s#Pclose_bracket", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["close_bracket"] = val
				fmt.Printf("[DEBUG] Found mapping for close_bracket -> %s\n", val)
			case fmt.Sprintf(":!%s#Pdelete_or_backspace", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["delete_or_backspace"] = val
				fmt.Printf("[DEBUG] Found mapping for delete_or_backspace -> %s\n", val)
			case fmt.Sprintf(":!%s#Preturn_or_enter", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["return_or_enter"] = val
				fmt.Printf("[DEBUG] Found mapping for return_or_enter -> %s\n", val)
			case fmt.Sprintf(":!%s#Pright_shift", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["right_shift"] = val
				fmt.Printf("[DEBUG] Found mapping for right_shift -> %s\n", val)
			case fmt.Sprintf(":!%s#Pright_option", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["right_option"] = val
				fmt.Printf("[DEBUG] Found mapping for right_option -> %s\n", val)
			case fmt.Sprintf(":!%s#Pright_command", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["right_command"] = val
				fmt.Printf("[DEBUG] Found mapping for right_command -> %s\n", val)
			case fmt.Sprintf(":!%s#Pspacebar", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["spacebar"] = val
				fmt.Printf("[DEBUG] Found mapping for spacebar -> %s\n", val)
			case fmt.Sprintf(":!%s#Pleft_arrow", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["left_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for left_arrow -> %s\n", val)
			case fmt.Sprintf(":!%s#Pright_arrow", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["right_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for right_arrow -> %s\n", val)
			case fmt.Sprintf(":!%s#Pup_arrow", TC):
				val := formatEdnValue(value)
				config.SpecialKeys["up_arrow"] = val
				fmt.Printf("[DEBUG] Found mapping for up_arrow -> %s\n", val)
			case fmt.Sprintf(":!%s#Pdown_arrow", TC):
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
	// If it's a vector, process each element.
	case []interface{}:
		var parts []string
		for _, item := range v {
			// If numeric, log and convert to string.
			switch num := item.(type) {
			case int, int8, int16, int32, int64, float32, float64:
				s := fmt.Sprintf("%v", num)
				fmt.Printf("[DEBUG] Converting numeric value %v -> %s\n", num, s)
				parts = append(parts, s)
			case edn.Keyword:
				str := string(num)
				if strings.HasPrefix(str, ":!") {
					parts = append(parts, strings.TrimPrefix(str, ":!"))
				} else {
					parts = append(parts, str)
				}
			default:
				parts = append(parts, fmt.Sprint(item))
			}
		}
		return strings.Join(parts, " ")
	case edn.Keyword:
		str := string(v)
		if strings.HasPrefix(str, ":!") {
			return strings.TrimPrefix(str, ":!")
		}
		return str
	// Explicitly handle numeric types.
	case int, int8, int16, int32, int64, float32, float64:
		s := fmt.Sprintf("%v", v)
		fmt.Printf("[DEBUG] Converting numeric value %v -> %s\n", v, s)
		return s
	default:
		return fmt.Sprint(v)
	}
}

func extractMappingComments(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	rawText := string(data)
	lines := strings.Split(rawText, "\n")
	var comments []string
	for _, line := range lines {
		// Look for rule lines (assume containing "[:!TC#P").
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

	// Layout string with 40 placeholders (note "CTRL" renamed to "CTL").
	layout := "┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬───────────┐\n" +
		"| ~ " + "`" + " | ! 1 | @ 2 | # 3 | $ 4 | %% 5 | ^ 6 | & 7 | * 8 | ( 9 | ) 0 | _ - | + = | %s |\n" +
		"| TAB | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n" +
		"| CAPS | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |      %s      |\n" +
		"| SHIFT  | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |     %s     |\n" +
		"| CTL | ALT | CMD │               %s               │ %s | %s │\n" +
		"└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴───────────┘\n"

	// Format the layout using its 40 placeholders.
	formattedLayout := fmt.Sprintf(layout,
		center(config.SpecialKeys["delete_or_backspace"], 8),
		center(config.Letters["q"], 3), center(config.Letters["w"], 3), center(config.Letters["e"], 3),
		center(config.Letters["r"], 3), center(config.Letters["t"], 3), center(config.Letters["y"], 3),
		center(config.Letters["u"], 3), center(config.Letters["i"], 3), center(config.Letters["o"], 3),
		center(config.Letters["p"], 3), config.SpecialKeys["open_bracket"], config.SpecialKeys["close_bracket"],
		center(config.SpecialKeys["backslash"], 8),
		center(config.Letters["a"], 3), center(config.Letters["s"], 3), center(config.Letters["d"], 3),
		center(config.Letters["f"], 3), center(config.Letters["g"], 3), center(config.Letters["h"], 3),
		center(config.Letters["j"], 3), center(config.Letters["k"], 3), center(config.Letters["l"], 3),
		config.SpecialKeys["semicolon"], config.SpecialKeys["quote"], center(config.SpecialKeys["return_or_enter"], 8),
		center(config.Letters["z"], 3), center(config.Letters["x"], 3), center(config.Letters["c"], 3),
		center(config.Letters["v"], 3), center(config.Letters["b"], 3), center(config.Letters["n"], 3),
		center(config.Letters["m"], 3), config.SpecialKeys["comma"], config.SpecialKeys["period"],
		config.SpecialKeys["slash"], center(config.SpecialKeys["right_shift"], 8),
		center(config.SpecialKeys["spacebar"], 16), config.SpecialKeys["right_command"],
		config.SpecialKeys["right_option"],
	)

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

	finalContent := markdownStart + codeFenceStart + formattedLayout + codeFenceEnd +
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
