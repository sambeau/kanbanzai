package storage

import "errors"

// ErrConflict is returned when a write fails because the file on disk
// has changed since the record was loaded (optimistic-locking violation).
var ErrConflict = errors.New("concurrent modification: file changed since last read")
