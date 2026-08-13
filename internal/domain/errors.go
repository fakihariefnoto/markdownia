// Package domain holds the entity types and domain errors. It has no
// dependencies on any other package in the tree.
package domain

import "errors"

// Sentinel domain errors. They are wrapped with context as they propagate and
// mapped to {code, content} by the binding layer. The zero value of each
// error's code string is its name; see binding/errors.go.
var (
	ErrSourceNotFound     = errors.New("source not found")
	ErrSourceUnavailable  = errors.New("source unavailable")
	ErrSourceBusy         = errors.New("source has an operation in progress")
	ErrPathEscapesRoot    = errors.New("path escapes source root")
	ErrNotAMarkdownFile   = errors.New("not a markdown file")
	ErrCloneFailed        = errors.New("git clone failed")
	ErrPullFailed         = errors.New("git pull failed")
	ErrAuthRequired       = errors.New("git authentication required")
	ErrRepoNotFound       = errors.New("git repository not found")
	ErrHostUnreachable    = errors.New("git host unreachable")
	ErrNotAGitRepo        = errors.New("not a git repository")
	ErrZipTooLarge        = errors.New("archive exceeds size limit")
	ErrZipCorrupt         = errors.New("archive is corrupt")
	ErrDocumentNotFound   = errors.New("document not found")
	ErrInvalidAnchor      = errors.New("invalid highlight anchor")
	ErrInvalidColor       = errors.New("invalid highlight color")
	ErrDuplicateName      = errors.New("name already exists")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrAssetNotFound      = errors.New("asset not found")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrOperationCancelled = errors.New("operation cancelled")
	ErrExportTargetEmpty  = errors.New("export target is empty")
	ErrRefreshNotAllowed  = errors.New("this source kind cannot be refreshed")
)
