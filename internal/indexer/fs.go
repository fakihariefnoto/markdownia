package indexer

import (
	"os"
)

// DirEntry mirrors os.DirEntry for the walker's testable indirection.
type DirEntry interface {
	Name() string
	IsDir() bool
}

// osReadDir wraps os.ReadDir, adapting to the DirEntry interface.
func osReadDir(dir string) ([]DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]DirEntry, len(entries))
	for i, e := range entries {
		out[i] = e
	}
	return out, nil
}
