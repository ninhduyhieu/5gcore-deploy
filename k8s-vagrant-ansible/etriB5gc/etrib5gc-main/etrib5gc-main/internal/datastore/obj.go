package datastore

type ObjectWrapper[T any] struct {
	key    string
	encode func(*T) ([]byte, error)
	decode func([]byte, *T) error
	obj    *T
}

func NewObjectLoader[T any](key string, encode func(*T) ([]byte, error), decode func([]byte, *T) error) *ObjectWrapper[T] {
	return &ObjectWrapper[T]{
		key:    key,
		encode: encode,
		decode: decode,
	}
}

func NewObjectWriter[T any](key string, obj *T, encode func(*T) ([]byte, error)) *ObjectWrapper[T] {
	return &ObjectWrapper[T]{
		key:    key,
		obj:    obj,
		encode: encode,
	}
}
