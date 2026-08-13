// Package binding is the Wails-bound service surface — the app's typed
// contract with the frontend (replacing the OpenAPI spec a networked backend
// would commit). Methods take ctx first, return (T, error), and never accept a
// filesystem path originating from page content.
package binding

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anofac/markdownia/internal/domain"
)

// Message is the standard {code, content} envelope the frontend's binding
// wrapper normalizes errors into.
type Message struct {
	Code    string `json:"code"`
	Content string `json:"content"`
}

// MapError converts a domain error into a user-safe {code, content} message.
// It is the reference implementation of the frontend error normalization; the
// Wails layer surfaces it alongside the returned error so the UI can render
// the same shape every backend call produces.
// No raw Go error text ever reaches the UI: an unmapped error is a bug, not a
// fallback — it returns a generic message and logs the full error.
func MapError(err error, logger *slog.Logger) Message {
	if err == nil {
		return Message{}
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Domain sentinels → code + user-facing content.
	switch {
	case errors.Is(err, domain.ErrSourceNotFound):
		return Message{Code: "source_not_found", Content: "The source no longer exists. It may have been deleted."}
	case errors.Is(err, domain.ErrSourceUnavailable):
		return Message{Code: "source_unavailable", Content: "This source is unavailable. Its files may have moved — try relocating it."}
	case errors.Is(err, domain.ErrSourceBusy):
		return Message{Code: "source_busy", Content: "A scan or import is already running for this source."}
	case errors.Is(err, domain.ErrPathEscapesRoot):
		return Message{Code: "path_escapes_root", Content: "This file lies outside the source's folder and cannot be opened."}
	case errors.Is(err, domain.ErrNotAMarkdownFile):
		return Message{Code: "not_markdown", Content: "This is not a markdown file."}
	case errors.Is(err, domain.ErrAuthRequired):
		return Message{Code: "git_auth_required", Content: "This repository requires authentication. Import a local folder instead, or check your credentials."}
	case errors.Is(err, domain.ErrRepoNotFound):
		return Message{Code: "git_not_found", Content: "No repository was found at that address. Check the URL."}
	case errors.Is(err, domain.ErrHostUnreachable):
		return Message{Code: "git_unreachable", Content: "Could not reach the repository host. Check your connection and try again."}
	case errors.Is(err, domain.ErrNotAGitRepo):
		return Message{Code: "git_not_repo", Content: "That folder is not a git repository."}
	case errors.Is(err, domain.ErrCloneFailed):
		return Message{Code: "git_clone_failed", Content: "The repository could not be cloned."}
	case errors.Is(err, domain.ErrPullFailed):
		return Message{Code: "git_pull_failed", Content: "The refresh could not pull the latest changes. Your existing copy is untouched."}
	case errors.Is(err, domain.ErrZipTooLarge):
		return Message{Code: "zip_too_large", Content: "The archive is too large to extract."}
	case errors.Is(err, domain.ErrZipCorrupt):
		return Message{Code: "zip_corrupt", Content: "The archive is corrupt or is not a valid zip file."}
	case errors.Is(err, domain.ErrDocumentNotFound):
		return Message{Code: "document_not_found", Content: "This document no longer exists in the library."}
	case errors.Is(err, domain.ErrInvalidAnchor):
		return Message{Code: "invalid_anchor", Content: "Select within a single paragraph to highlight."}
	case errors.Is(err, domain.ErrInvalidColor):
		return Message{Code: "invalid_color", Content: "That highlight color is not supported."}
	case errors.Is(err, domain.ErrDuplicateName):
		return Message{Code: "duplicate_name", Content: "That name is already in use."}
	case errors.Is(err, domain.ErrCollectionNotFound):
		return Message{Code: "collection_not_found", Content: "This collection no longer exists."}
	case errors.Is(err, domain.ErrAssetNotFound):
		return Message{Code: "asset_not_found", Content: "The image could not be found on disk."}
	case errors.Is(err, domain.ErrInvalidArgument):
		return Message{Code: "invalid_argument", Content: "That input is not valid."}
	case errors.Is(err, domain.ErrOperationCancelled):
		return Message{Code: "cancelled", Content: "The operation was cancelled."}
	case errors.Is(err, domain.ErrExportTargetEmpty):
		return Message{Code: "export_empty", Content: "There are no documents to export."}
	case errors.Is(err, domain.ErrRefreshNotAllowed):
		return Message{Code: "refresh_not_allowed", Content: "Zip sources cannot be refreshed — their original archive is gone."}
	}

	// Unmapped: log the full error, surface a generic message. This is a bug.
	logger.Error("unmapped domain error", "error", err)
	return Message{Code: "error", Content: "Something went wrong. See Help → Reveal Log File for details."}
}

// ctxFromArgs extracts a context (Wails passes a real one; tests pass nil).
func ctxFromArgs(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
