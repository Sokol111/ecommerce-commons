package mongo

type CollectionWrapper[T any] struct {
	Coll T
}
