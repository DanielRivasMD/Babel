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
	"github.com/BurntSushi/toml"
	"github.com/DanielRivasMD/horus"
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

// TODO: redundant functions?
func loadFormat(path string) map[string]map[string]string {
	var cfg map[string]map[string]string
	_, err := toml.DecodeFile(path, &cfg)

	horus.CheckErr(
		err,
		horus.WithMessage("failed to load format config from: "+path),
		horus.WithExitCode(2),
		horus.WithFormatter(func(he *horus.Herror) string {
			return horus.OneLineErr(he.Message)
		}),
	)

	return cfg
}

////////////////////////////////////////////////////////////////////////////////////////////////////
