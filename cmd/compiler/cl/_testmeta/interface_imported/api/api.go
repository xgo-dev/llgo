package api

type Reader interface {
	Read([]byte) (int, error)
}

type Source struct{}

func (Source) Read([]byte) (int, error) {
	return 7, nil
}

func (Source) Close() error {
	return nil
}
