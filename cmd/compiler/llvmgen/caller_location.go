package llvmgen

import "github.com/xgo-dev/llgo/cmd/compiler/callerlocation"

// EmitLocationRecord emits an idempotent runtime source-location update unless
// the current block already has the same update and debug position with no
// intervening call or terminator. emit returns the update it appended;
// any instructions following it remain visible to the tracker.
// Non-tail insertion keeps all records; it cannot reuse append-only history.
func (b Builder) EmitLocationRecord(kind, name, file string, line int, emit func() Expr) {
	if !b.locationRecordsAtEnd {
		emit()
		return
	}
	block := b.impl.GetInsertBlock()
	location := callerlocation.Location{Kind: kind, Function: name, File: file, Line: line}
	if b.locationRecords.Repeated(block, block.LastInstruction(), location, b.diLocation) {
		return
	}
	b.locationRecords.Checkpoint(emit().impl)
}
