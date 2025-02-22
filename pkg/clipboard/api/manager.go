package api

type Manager interface {
	Get() ([]byte, error)
	Set(data []byte) error
}
