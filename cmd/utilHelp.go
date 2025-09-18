////////////////////////////////////////////////////////////////////////////////////////////////////

package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"github.com/DanielRivasMD/domovoi"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var helpRoot = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Interpret hotkeys into markdown keyboard visuals",
)

var helpDisplay = domovoi.FormatHelp(
	"Daniel Rivas",
	"danielrivasmd@gmail.com",
	"Scan EDN metadata & output Markdown table",
)

var helpInterpret = domovoi.FormatHelp(
	"Daniel Rivas",
	"<danielrivasmd@gmail.com>",
	"Load EDN metadata & produce configs",
)

var helpConstruct = domovoi.FormatHelp(
	"Daniel Rivas",
	"<danielrivasmd@gmail.com>",
	"Install configs & create paths",
)

var helpEmbed = domovoi.FormatHelp(
	"Daniel Rivas",
	"<danielrivasmd@gmail.com>",
	"Inserting key sequences over templates",
)

////////////////////////////////////////////////////////////////////////////////////////////////////
