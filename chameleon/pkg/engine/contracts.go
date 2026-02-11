package engine

type MutationType int

const (
	MutationInsert MutationType = iota
	MutationUpdate
	MutationDelete
)

type Mutation struct {
	Type         MutationType
	Entity       string
	HasWhere     bool
	AffectedRows int64
}

// MutationBuilder builds and executes a mutation
//
// Implementations live outside the engine package (e.g. pkg/mutation)
type MutationBuilder interface {
	// Build resolves the mutation into a generic structure
	Build() (*Mutation, error)

	// Exec executes the mutation against the database
	Exec() error
}

// MutationFactory creates mutation builders for the engine
//
// The engine depends ONLY on this interface, never on concrete implementations.
type MutationFactory interface {
	Insert(entity string) MutationBuilder
	Update(entity string) MutationBuilder
	Delete(entity string) MutationBuilder
}
