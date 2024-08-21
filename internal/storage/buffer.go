package storage

type InMemory struct {
	Data []byte
}

func (b *InMemory) Read() ([]byte, error) {
	if len(b.Data) > 0 {
		return b.Data, nil
	}
	return nil, ErrNoCertFound
}

func (b *InMemory) CanRead() error {
	if len(b.Data) > 0 {
		return nil
	}

	return ErrNoCertFound
}

func (b *InMemory) Write(data []byte) error {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	b.Data = data
	return nil
}

func (b *InMemory) CanWrite() error {
	return nil
}
