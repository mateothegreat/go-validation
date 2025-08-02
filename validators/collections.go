package validators

type OneOf[T comparable] struct {
	values []T
}

func NewOneOf[T comparable](values ...T) *OneOf[T] {
	return &OneOf[T]{values: values}
}

// func (o *OneOf[T]) Validate(field string, value T, rule string) error {
