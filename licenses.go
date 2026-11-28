// Package survivalgame holds the files that ship beside the game and are
// not Go code.
package survivalgame

import _ "embed"

// Licenses is LICENSES.txt: the licences of every font, recording, library and
// recipe the game is built from, word for word. It is made by
// `go run ./cmd/licenses`. The game prints it with `-licenses`, and a release
// ships the file beside the exe.
//
//go:embed LICENSES.txt
var Licenses string
