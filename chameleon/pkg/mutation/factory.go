package mutation

import "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"

// ============================================================
// MUTATION FACTORY
// ============================================================
//

type Factory struct {
	schema *engine.Schema
}

func NewFactory(schema *engine.Schema) *Factory {
	return &Factory{schema: schema}
}

func (f *Factory) Insert(entity string) engine.MutationBuilder {
	return NewInsertBuilder(f.schema, entity)
}

func (f *Factory) Update(entity string) engine.MutationBuilder {
	return NewUpdateBuilder(f.schema, entity)
}

func (f *Factory) Delete(entity string) engine.MutationBuilder {
	return NewDeleteBuilder(f.schema, entity)
}
