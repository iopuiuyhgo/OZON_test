package storage

type Storage interface {
	Load(key string) (value string, ok bool)
	Store(key string, value string)
}
