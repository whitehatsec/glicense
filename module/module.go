package module

import (
	"fmt"
)

// Module represents a single Go module.
//
// Depending on the source that this is parsed from, fields may be empty.
// All helper functions on Module work with zero values. See their associated
// documentation for more information on exact behavior.
type Module struct {
	Path     string // Import path, such as "github.com/hpapaxen/glicense"
	Version  string // Version like "v1.2.3"
	Indirect bool
	Hash     string // Hash such as "h1:abcd1234"
}

// String returns a human readable string format.
func (m *Module) String() string {
	return fmt.Sprintf("%s (%s)", m.Path, m.Version)
}
