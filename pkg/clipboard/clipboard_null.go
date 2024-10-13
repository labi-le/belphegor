package clipboard

type Null struct {
	data []byte
}

func (n Null) Get() ([]byte, error) {
	return n.data, nil
}

func (n Null) Set(data []byte) error {
	n.data = data
	return nil
}

func (n Null) Name() string {
	return NullClipboard
}
