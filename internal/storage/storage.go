package storage

type Storage interface {
	Load(key string) (string, error)
	Store(key string, value string) error
}
