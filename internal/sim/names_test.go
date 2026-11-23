package sim

import "testing"

// Every kind shows a name in labels, notices and the bag; a new kind
// appended without one shows an empty label.
func TestEveryKindHasAName(t *testing.T) {
	for k := EntityKind(1); k < NumEntityKinds; k++ {
		if n := k.Name(); n == "" || n == entityNames[EntityNone] {
			t.Errorf("entity kind %d has no name", k)
		}
	}
	for k := ItemKind(1); k < NumItems; k++ {
		if k.Def().Name == "" {
			t.Errorf("item kind %d has no name", k)
		}
	}
}
