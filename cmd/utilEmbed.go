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
	"log"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func embedConfig(entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	switch {

	case target == "kanata":
		// List of programs whose bindings we want to use for kanata remapping.
		allowedPrograms := map[string]bool{
			"serpl": true,
		}

		replaces := []moldReplace{}
		for _, entry := range entries {
			// Check if any action belongs to an allowed program
			var hasAllowed bool
			for _, act := range entry.Actions {
				if allowedPrograms[act.Program] {
					hasAllowed = true
					break
				}
			}
			if !hasAllowed {
				continue
			}

			// fmt.Println(entry.Trigger.Key)
			// fmt.Println(entry.Trigger.Mode)
			// fmt.Println(entry.Trigger.Modifier)
			// Format the trigger (input) using kanata's lookup and empty separator
			// triggerKey := formatKeySeq(entry.Trigger, lookups.embed, "kanata", "")
			// if triggerKey == "" {
			// 	continue
			// }

			triggerKey := entry.Trigger.Modifier + entry.Trigger.Key
			if triggerKey != "Oq" {
				continue
			}
			fmt.Println(triggerKey)

			// Format the binding (output) similarly
			bindKey := formatKeySeq(entry.Binding, lookups.embed, "kanata", "")
			if bindKey == "" {
				continue
			}
			fmt.Println(bindKey)

			bindKey = "A-q"

			// Construct the line as it appears in the compose template:
			// "  {trigger}      XX"
			linePrefix := fmt.Sprintf("  %s", triggerKey)
			padding := 10 - len(linePrefix)
			if padding < 1 {
				padding = 1
			}
			oldLine := triggerKey
			newLine := linePrefix + strings.Repeat(" ", padding) + bindKey + ":line"

			replaces = append(replaces, replace(oldLine, newLine))
		}

		// TODO: handle sequence annotations (chords, multiple triggers)
		if len(replaces) == 0 {
			log.Printf("Warning: No kanata bindings found for allowed programs")
		}
		mf := newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...)
		moldForging("embed-kanata", mf)

	case target == "serpl":
		embedBindings(entries, target, func(key, val string) string {
			x := fmt.Sprintf("\\\"<%s>\\\" = \\\"%s\\\":line", key, val)
			println(x)
			return fmt.Sprintf("\\\"<%s>\\\" = \\\"%s\\\":line", key, val)
		})

	// case target == "lazygit":
	// 	embedBindings(entries, target, func(key, val string) string {
	// 		return fmt.Sprintf("    %s: '<%s>':line", val, key)
	// 	})

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

		// TODO: add horus error handler
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
	fmt.Println(mf)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
