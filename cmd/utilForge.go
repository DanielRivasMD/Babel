////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"strings"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
)

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

func newMoldConfig(
	outFile string,
	inFiles []string,
	replaces ...moldReplace,
) moldForge {
	return moldForge{
		out:      outFile,
		files:    inFiles,
		replaces: replaces,
	}
}

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
