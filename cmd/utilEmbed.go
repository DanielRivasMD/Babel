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
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/DanielRivasMD/horus"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func embedConfig(entries []BindingEntry, target string) {
	filtered := filterByProgram(entries, target)

	switch {

	case target == "kanata":
		allowedPrograms := map[string]bool{
			"helix":   true,
			"serpl":   true,
			"lazygit": true,
			"zellij":  true,
			"term":    true,
			"micro":   true,
			"kanata":  true,
		}

		// TODO: add condition for entries with sequence present
		replaces := []moldReplace{}
		for _, entry := range entries {
			hasAllowed := false
			for _, act := range entry.Actions {
				normProgram := normalizeProgram(act.Program)
				if allowedPrograms[normProgram] {
					hasAllowed = true
					break
				}
			}
			if !hasAllowed {
				continue
			}

			triggerKey := formatTriggerEntry(entry.Trigger, lookups.embed, "kanata", triggerTransforms)
			if triggerKey == "" {
				continue
			}

			bindKey := formatKeySeq(entry.Binding, lookups.embed, "kanata", "")
			if bindKey == "" {
				continue
			}

			linePrefix := fmt.Sprintf("  %s", triggerKey)
			padding := 10 - len(linePrefix)
			if padding < 1 {
				padding = 1
			}
			oldLine := linePrefix
			newLine := linePrefix + strings.Repeat(" ", padding) + bindKey

			replaces = append(replaces, replace(oldLine, newLine+":line"))
		}

		if len(replaces) == 0 {
			log.Printf("Warning: No kanata bindings found for allowed programs")
		}
		mf := newMoldConfig(embedFlags.target, []string{embedFlags.target}, replaces...)
		moldForging("embed-kanata", mf)

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
		horus.CheckErr(
			errors.New(""),
			horus.WithMessage("unsupported --program: " + target),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string {
				return horus.OneLineErr(he.Message)
			}),
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

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
