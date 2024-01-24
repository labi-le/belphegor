package encryption

type Parts struct {
	Raw      [][]byte
	PartSize int
}

func NewParts(raw []byte, partSize int) *Parts {
	return &Parts{
		Raw:      extractParts(raw, partSize),
		PartSize: partSize,
	}
}

func extractParts(raw []byte, partSize int) [][]byte {
	if partSize <= 0 {
		panic("invalid part size")
	}

	partCount := len(raw) / partSize

	parts := make([][]byte, partCount)

	start := 0
	for i := 0; i < partCount; i++ {
		end := start + partSize
		if end > len(raw) {
			end = len(raw)
		}
		parts[i] = raw[start:end]
		start = end
	}

	return parts
}

func (p *Parts) EncryptSelf(fn func([]byte) ([]byte, error)) error {
	var err error
	for i := range p.Raw {
		if p.Raw[i], err = fn(p.Raw[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parts) DecryptSelf(fn func([]byte) ([]byte, error)) error {
	var err error
	for i := range p.Raw {
		if p.Raw[i], err = fn(p.Raw[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parts) Glue() []byte {
	totalSize := 0

	for i := range p.Raw {
		totalSize += len(p.Raw[i])
	}

	result := make([]byte, totalSize)
	offset := 0

	for i := range p.Raw {
		copy(result[offset:], p.Raw[i])
		offset += len(p.Raw[i])
	}

	return result
}
