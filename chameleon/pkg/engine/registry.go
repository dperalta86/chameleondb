// engine/registry.go
package engine

var mutationFactory MutationFactory

func RegisterMutationFactory(factory MutationFactory) {
	if mutationFactory != nil {
		panic("mutation factory already registered")
	}
	mutationFactory = factory
}

func getMutationFactory() MutationFactory {
	if mutationFactory == nil {
		panic("no mutation factory registered")
	}
	return mutationFactory
}
