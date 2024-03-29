package internal

// Channel is an interface for managing clipboard data.
type Channel interface {
	// Write data to the provided byte slice.
	Write(b []byte)

	// Read returns a read-only channel for retrieving clipboard data.
	Read() <-chan []byte
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
func (c *UpdateChannel) Write(b []byte) {
	c.raw <- b
}

// Get implements the Get method of the Channel interface.
func (c *UpdateChannel) Read() <-chan []byte {
	return c.raw
}
