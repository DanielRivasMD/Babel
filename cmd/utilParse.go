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

import (
	"fmt"
	"os"
	"strings"

	"olympos.io/encoding/edn"
)

// Configurable variables
const (
	TcPrefix   = "TC" // Change this to modify the EDN prefix
	DefaultKey = " "  // Display for unmapped keys
	OutputDir  = "layouts"
	OutputFile = "keyboard_layout.md"
	// Define the EDN config file location.
	// ednFile = "keyboard_config.edn"
)

type KeyboardConfig struct {
	Letters      map[string]string
	SpecialKeys  map[string]string
	UsedTcPrefix string
}

func parse() {
	// Optionally, you might call edn.DoStuff() here according to documentation.
	config := parseEdnConfig(ednFile)
	generateMarkdown(config)
	fmt.Printf("Generated layout using TC variable: '%s'\n", TcPrefix)
	fmt.Printf("Output: %s/%s\n", OutputDir, OutputFile)
}

func parseEdnConfig(filePath string) KeyboardConfig {
	// Read the file bytes.
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error reading EDN file: %v", err))
	}

	// Unmarshal into a map with edn.Keyword keys.
	var data map[edn.Keyword]interface{}
	if err := edn.Unmarshal(file, &data); err != nil {
		panic(fmt.Sprintf("Error parsing EDN: %v", err))
	}

	config := KeyboardConfig{
		Letters:     make(map[string]string),
		SpecialKeys: make(map[string]string),
	}

	// Initialize all letter keys (a-z) with the default display.
	for c := 'a'; c <= 'z'; c++ {
		config.Letters[string(c)] = DefaultKey
	}

	// Initialize special keys.
	specialKeys := []string{
		"open_bracket", "close_bracket", "semicolon", "quote",
		"backslash", "comma", "period", "slash",
		"delete_or_backspace", "return_or_enter",
		"right_shift", "right_option", "right_command", "spacebar",
		// Also initialize arrow keys.
		"left_arrow", "right_arrow", "up_arrow", "down_arrow",
	}
	for _, key := range specialKeys {
		config.SpecialKeys[key] = DefaultKey
	}

	// Set default labels for common special keys.
	config.SpecialKeys["delete_or_backspace"] = "BACK"
	config.SpecialKeys["return_or_enter"] = "ENTER"
	config.SpecialKeys["right_shift"] = "SHIFT"
	config.SpecialKeys["right_option"] = "ALT"
	config.SpecialKeys["right_command"] = "CMD"
	config.SpecialKeys["spacebar"] = "SPACE"

	// Parse EDN rules for custom mappings using the keyword key ":rules".
	rulesRaw, ok := data[edn.Keyword(":rules")]
	if !ok {
		return config
	}
	rules, ok := rulesRaw.([]interface{})
	if !ok {
		return config
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

		// Handle letter keys (a-z).
		for c := 'a'; c <= 'z'; c++ {
			letter := string(c)
			if keyStr == fmt.Sprintf(":!%s#P%s", TcPrefix, letter) {
				config.Letters[letter] = formatEdnValue(value)
				break
			}
		}

		// Handle special keys.
		switch keyStr {
		case fmt.Sprintf(":!%s#Popen_bracket", TcPrefix):
			config.SpecialKeys["open_bracket"] = formatEdnValue(value)
		case fmt.Sprintf(":!%s#Pclose_bracket", TcPrefix):
			config.SpecialKeys["close_bracket"] = formatEdnValue(value)
		case fmt.Sprintf(":!%s#Pdelete_or_backspace", TcPrefix):
			config.SpecialKeys["delete_or_backspace"] = formatEdnValue(value)
		// Arrow keys.
		case fmt.Sprintf(":!%s#Pleft_arrow", TcPrefix):
			config.SpecialKeys["left_arrow"] = formatEdnValue(value)
		case fmt.Sprintf(":!%s#Pright_arrow", TcPrefix):
			config.SpecialKeys["right_arrow"] = formatEdnValue(value)
		case fmt.Sprintf(":!%s#Pup_arrow", TcPrefix):
			config.SpecialKeys["up_arrow"] = formatEdnValue(value)
		case fmt.Sprintf(":!%s#Pdown_arrow", TcPrefix):
			config.SpecialKeys["down_arrow"] = formatEdnValue(value)
		}
	}

	config.UsedTcPrefix = TcPrefix
	return config
}

func formatEdnValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var parts []string
		for _, item := range v {
			// If the item is a keyword, we remove the prefix ":!T" if present.
			if kw, ok := item.(edn.Keyword); ok {
				str := string(kw)
				// For example, if str is ":!Tl", remove leading ":!T" to get "l"
				if strings.HasPrefix(str, ":!T") {
					parts = append(parts, strings.TrimPrefix(str, ":!T"))
				} else {
					parts = append(parts, str)
				}
			} else {
				parts = append(parts, fmt.Sprint(item))
			}
		}
		return strings.Join(parts, " ")
	case edn.Keyword:
		str := string(v)
		if strings.HasPrefix(str, ":!T") {
			return strings.TrimPrefix(str, ":!T")
		}
		return str
	default:
		return fmt.Sprint(v)
	}
}

// extractMappingComments processes the raw EDN file text to extract comment pairs.
// For rule lines that contain comments like "; close             ; helix",
// it produces a mapping "helix => close".
func extractMappingComments(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	rawText := string(data)
	lines := strings.Split(rawText, "\n")
	var comments []string
	for _, line := range lines {
		// Look for rule lines by detecting our rule pattern.
		if strings.Contains(line, "[:!TC#P") {
			// Split the line by semicolon.
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
	// Create output directory.
	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		panic(fmt.Sprintf("Error creating output directory: %v", err))
	}

	file, err := os.Create(fmt.Sprintf("%s/%s", OutputDir, OutputFile))
	if err != nil {
		panic(fmt.Sprintf("Error creating output file: %v", err))
	}
	defer file.Close()

	// Helper function to center text within a fixed width.
	center := func(text string, width int) string {
		if len(text) >= width {
			return text
		}
		padding := (width - len(text)) / 2
		return fmt.Sprintf("%*s%s%*s", padding, "", text, padding, "")
	}

	// Build the markdown header.
	markdownStart := fmt.Sprintf(`# Dynamic Keyboard Layout
*Generated with TC='%s'*

`, config.UsedTcPrefix)
	codeFenceStart := "```markdown\n"
	codeFenceEnd := "```\n"

	// The layout string with 40 placeholders.
	layout := "┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬───────────┐\n" +
		"| ~ " + "`" + " | ! 1 | @ 2 | # 3 | $ 4 | %% 5 | ^ 6 | & 7 | * 8 | ( 9 | ) 0 | _ - | + = | %s |\n" +
		"| TAB | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n" +
		"| CAPS | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |      %s      |\n" +
		"| SHIFT  | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |     %s     |\n" +
		"| CTRL | ALT | CMD │               %s               │ %s | %s │\n" +
		"└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴───────────┘\n"

	// First, format the layout string with its 40 placeholders.
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

	// Build the active mappings section.
	activeMappingsSection := fmt.Sprintf("\n### Active Mappings\n- **Letters**: %s\n- **Specials**: %s (SPACE), %s (ENTER)\n- **Arrows**: %s\n- **TC Variable**: '%s' (change in script)\n",
		getActiveMappings(config.Letters),
		config.SpecialKeys["spacebar"],
		config.SpecialKeys["return_or_enter"],
		getArrowMappings(config),
		config.UsedTcPrefix,
	)

	// Extract mapping comments from the EDN file.
	mappingComments := extractMappingComments(ednFile)
	mappingCommentsSection := ""
	if len(mappingComments) > 0 {
		mappingCommentsSection = "\n### Mapping Comments\n"
		for _, comment := range mappingComments {
			mappingCommentsSection += "- " + comment + "\n"
		}
	}

	// Assemble the final content.
	finalContent := markdownStart + codeFenceStart + formattedLayout + codeFenceEnd +
		activeMappingsSection +
		mappingCommentsSection

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
