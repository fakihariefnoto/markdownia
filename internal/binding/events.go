package binding

import (
	"github.com/anofac/markdownia/internal/domain"
)

// EventEmitter emits progress and status events to the frontend. The concrete
// implementation is injected at wiring time.
type EventEmitter interface {
	Emit(name string, data any)
}

// progressSink adapts the usecase ProgressSink interface to event emission.
type progressSink struct {
	emitter EventEmitter
}

// NewProgressSink wires the usecases' progress reporting to Wails events.
func NewProgressSink(emitter EventEmitter) *progressSink {
	return &progressSink{emitter: emitter}
}

// SourceProgressPayload is the source:progress event payload.
type SourceProgressPayload struct {
	SourceID int64  `json:"sourceId"`
	Phase    string `json:"phase"`
	Current  int    `json:"current"`
	Total    int    `json:"total"`
}

func (p *progressSink) SourceProgress(sourceID int64, phase string, current, total int) {
	p.emitter.Emit("source:progress", SourceProgressPayload{
		SourceID: sourceID, Phase: phase, Current: current, Total: total,
	})
}

// SourceStatusPayload is the source:status event payload.
type SourceStatusPayload struct {
	SourceID    int64  `json:"sourceId"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

func (p *progressSink) SourceStatus(sourceID int64, status domain.SourceStatus, errMsg string) {
	p.emitter.Emit("source:status", SourceStatusPayload{
		SourceID: sourceID, Status: string(status), Error: errMsg,
	})
}

// SourceIndexedPayload is the source:indexed event payload.
type SourceIndexedPayload struct {
	SourceID         int64 `json:"sourceId"`
	Indexed          int64 `json:"indexed"`
	RemovedHighlights int64 `json:"removedHighlights"`
	DeletedDocs      int64 `json:"deletedDocs"`
}

func (p *progressSink) SourceIndexed(sourceID int64, indexed, removedHighlights, deletedDocs int64) {
	p.emitter.Emit("source:indexed", SourceIndexedPayload{
		SourceID: sourceID, Indexed: indexed,
		RemovedHighlights: removedHighlights, DeletedDocs: deletedDocs,
	})
}

func (p *progressSink) SearchInvalidated() {
	p.emitter.Emit("search:invalidated", map[string]any{})
}
