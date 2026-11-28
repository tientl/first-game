package sim

import (
	"strings"
	"testing"
)

// Every trophy head is named after its beast, however late in the package's
// init order that beast is registered.
func TestHeadNames(t *testing.T) {
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		h := HeadItem(k)
		if h == ItemNone {
			continue
		}
		name := h.Def().Name
		if strings.TrimSpace(strings.TrimPrefix(name, "Đầu")) == "" {
			t.Errorf("head of creature %d is named %q", k, name)
		}
	}
}
