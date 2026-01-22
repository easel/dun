package builtin

import (
	"embed"
	"io/fs"
)

//go:embed helix/** git/**
var builtinFS embed.FS

type Entry struct {
	ID   string
	FS   fs.FS
	Base string
}

func Plugins() []Entry {
	return []Entry{
		{
			ID:   "helix",
			FS:   builtinFS,
			Base: "helix",
		},
		{
			ID:   "git",
			FS:   builtinFS,
			Base: "git",
		},
	}
}
