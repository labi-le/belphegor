package storage

// Storage represents a storage for storing nodes.
type Storage[key any, val any] interface {
	// Add adds the specified node to the storage.
	// If the node already exists, it will be overwritten.
	Add(k key, v val)
	// Delete deletes the node associated with the specified id.
	Delete(k key)
	// Get returns the node associated with the specified id.
	Get(k key) (val, bool)
	// Exist returns true if the specified node exists in the storage.
	Exist(k key) bool
	// Tap calls the specified function for parallel each node in the storage.
	Tap(fn func(key, val) bool)
}
