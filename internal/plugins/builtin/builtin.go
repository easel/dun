package builtin

import (
	"embed"
	"io/fs"
)

//go:embed helix/**
var helixFS embed.FS

type Entry struct {
	ID   string
	FS   fs.FS
	Base string
}

func Plugins() []Entry {
	return []Entry{
		{
			ID:   "helix",
			FS:   helixFS,
			Base: "helix",
		},
	}
}
