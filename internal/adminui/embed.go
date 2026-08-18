package adminui

import (
	"embed"
	"io/fs"
)

// Files is the go:embed tree of the production Vite build under dist/.
// The committed dist is required so go test / go run ./examples/admin work
// without npm at compile time. Consumers go get the Gombit module and the
// admin SPA is already inside the binary.
//
//go:embed all:dist
var Files embed.FS

// FS returns the admin SPA tree rooted at dist/. If fs.Sub fails, it
// returns an empty embed.FS rather than Files: the SPA lives under dist/,
// so the un-subbed root would miss index.html and skip mount.
func FS() fs.FS {
	sub, err := fs.Sub(Files, "dist")
	if err != nil {
		return embed.FS{}
	}
	return sub
}
