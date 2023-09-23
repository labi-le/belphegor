package belphegor

type Channel interface {
	Set(b []byte)
	Get() <-chan []byte
}
type UpdateChannel struct {
	raw chan []byte
}

func NewChannel() Channel {
	return &UpdateChannel{raw: make(chan []byte)}
}

func (c *UpdateChannel) Set(b []byte) {
	c.raw <- b
}

func (c *UpdateChannel) Get() <-chan []byte {
	return c.raw
}
