package belphegor

// Channel is an interface for managing clipboard data.
type Channel interface {
	// Set sets the clipboard data to the provided byte slice.
	Set(b []byte)

	// Get returns a read-only channel for retrieving clipboard data.
	Get() <-chan []byte
}

// UpdateChannel is an implementation of the Channel interface.
type UpdateChannel struct {
	raw chan []byte
}

// NewChannel creates a new instance of UpdateChannel and returns it as a Channel interface.
func NewChannel() *UpdateChannel {
	return &UpdateChannel{raw: make(chan []byte)}
}

// Set implements the Set method of the Channel interface.
func (c *UpdateChannel) Set(b []byte) {
	c.raw <- b
}

// Get implements the Get method of the Channel interface.
func (c *UpdateChannel) Get() <-chan []byte {
	return c.raw
}
