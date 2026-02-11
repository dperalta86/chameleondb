package mutation

// ============================================================
// MUTATION FACTORY NOW LIVES IN engine/registry.go
// ============================================================
//

/* type Factory struct {
	schema *engine.Schema
}

func NewFactory(schema *engine.Schema) *Factory {
	return &Factory{schema: schema}
}

func (f *Factory) Insert(entity string) engine.InsertMutation {
	return NewInsertBuilder(f.schema, entity)
}

func (f *Factory) Update(entity string) engine.UpdateMutation {
	return NewUpdateBuilder(f.schema, entity)
}

func (f *Factory) Delete(entity string) engine.DeleteMutation {
	return NewDeleteBuilder(f.schema, entity)
}
*/
