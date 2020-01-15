// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collection

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
)

// Schemas contains metadata about configuration resources.
type Schemas struct {
	byCollection map[Name]Schema
	byAddOrder   []Schema
}

// SchemasFor is a shortcut for creating Schemas. It uses MustAdd for each element.
func SchemasFor(schemas ...Schema) Schemas {
	b := NewSchemasBuilder()
	for _, s := range schemas {
		b.MustAdd(s)
	}
	return b.Build()
}

// SchemasBuilder is a builder for the schemas type.
type SchemasBuilder struct {
	schemas Schemas
}

// NewSchemasBuilder returns a new instance of SchemasBuilder.
func NewSchemasBuilder() *SchemasBuilder {
	s := Schemas{
		byCollection: make(map[Name]Schema),
	}

	return &SchemasBuilder{
		schemas: s,
	}
}

// Add a new collection to the schemas.
func (b *SchemasBuilder) Add(s Schema) error {
	if _, found := b.schemas.byCollection[s.Name()]; found {
		return fmt.Errorf("collection already exists: %v", s.Name())
	}

	b.schemas.byCollection[s.Name()] = s
	b.schemas.byAddOrder = append(b.schemas.byAddOrder, s)
	return nil
}

// MustAdd calls Add and panics if it fails.
func (b *SchemasBuilder) MustAdd(s Schema) *SchemasBuilder {
	if err := b.Add(s); err != nil {
		panic(fmt.Sprintf("SchemasBuilder.MustAdd: %v", err))
	}
	return b
}

// Build a new schemas from this SchemasBuilder.
func (b *SchemasBuilder) Build() Schemas {
	s := b.schemas

	// Avoid modify after Build.
	b.schemas = Schemas{}

	return s
}

// ForEach executes the given function on each contained schema, until the function returns true.
func (s Schemas) ForEach(handleSchema func(Schema) (done bool)) {
	for _, schema := range s.byAddOrder {
		if handleSchema(schema) {
			return
		}
	}
}

// Find looks up a Schema by its collection name.
func (s Schemas) Find(collection string) (Schema, bool) {
	i, ok := s.byCollection[Name(collection)]
	return i, ok
}

// MustFind calls Find and panics if not found.
func (s Schemas) MustFind(collection string) Schema {
	i, ok := s.Find(collection)
	if !ok {
		panic(fmt.Sprintf("schemas.MustFind: matching entry not found for collection: %q", collection))
	}
	return i
}

// FindByKind searches and returns the first schema with the given kind
func (s Schemas) FindByKind(kind string) (Schema, bool) {
	for _, rs := range s.byAddOrder {
		if strings.EqualFold(rs.Resource().Kind(), kind) {
			return rs, true
		}
	}

	return nil, false
}

// MustFindByKind calls FindByKind and panics if not found.
func (s Schemas) MustFindByKind(kind string) Schema {
	r, found := s.FindByKind(kind)
	if !found {
		panic(fmt.Sprintf("Schemas.MustFindByKind: unable to find %s", kind))
	}
	return r
}

// FindByGroupAndKind searches and returns the first schema with the given group/kind
func (s Schemas) FindByGroupAndKind(group, kind string) (Schema, bool) {
	for _, rs := range s.byAddOrder {
		if rs.Resource().Group() == group && strings.EqualFold(rs.Resource().Kind(), kind) {
			return rs, true
		}
	}

	return nil, false
}

// MustFind calls FindByGroupAndKind and panics if not found.
func (s Schemas) MustFindByGroupAndKind(group, kind string) Schema {
	r, found := s.FindByGroupAndKind(group, kind)
	if !found {
		panic(fmt.Sprintf("Schemas.MustFindByGroupAndKind: unable to find %s/%s", group, kind))
	}
	return r
}

// All returns all known Schemas
func (s Schemas) All() []Schema {
	return append(make([]Schema, 0, len(s.byAddOrder)), s.byAddOrder...)
}

// Add creates a copy of this Schemas with the given schemas added.
func (s Schemas) Add(toAdd ...Schema) Schemas {
	b := NewSchemasBuilder()

	for _, s := range s.byAddOrder {
		b.MustAdd(s)
	}

	for _, s := range toAdd {
		b.MustAdd(s)
	}

	return b.Build()

}

// Remove creates a copy of this Schemas with the given schemas removed.
func (s Schemas) Remove(toRemove ...Schema) Schemas {
	b := NewSchemasBuilder()

	for _, s := range s.byAddOrder {
		shouldAdd := true
		for _, r := range toRemove {
			if r.Name() == s.Name() {
				shouldAdd = false
				break
			}
		}
		if shouldAdd {
			b.MustAdd(s)
		}
	}

	return b.Build()
}

// CollectionNames returns all known collections.
func (s Schemas) CollectionNames() Names {
	result := make(Names, 0, len(s.byAddOrder))

	for _, info := range s.byAddOrder {
		result = append(result, info.Name())
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.Compare(result[i].String(), result[j].String()) < 0
	})

	return result
}

// Kinds returns all known resource kinds.
func (s Schemas) Kinds() []string {
	kinds := make(map[string]struct{}, len(s.byAddOrder))
	for _, s := range s.byAddOrder {
		kinds[s.Resource().Kind()] = struct{}{}
	}

	out := make([]string, 0, len(kinds))
	for kind := range kinds {
		out = append(out, kind)
	}

	sort.Strings(out)
	return out
}

// DisabledCollectionNames returns the names of disabled collections
func (s Schemas) DisabledCollectionNames() Names {
	disabledCollections := make(Names, 0)
	for _, i := range s.byAddOrder {
		if i.IsDisabled() {
			disabledCollections = append(disabledCollections, i.Name())
		}
	}
	return disabledCollections
}

// Validate the schemas. Returns error if there is a problem.
func (s Schemas) Validate() (err error) {
	for _, c := range s.byAddOrder {
		err = multierror.Append(err, c.Resource().Validate()).ErrorOrNil()
	}
	return
}

func (s Schemas) Equal(o Schemas) bool {
	return cmp.Equal(s.byAddOrder, o.byAddOrder)
}
