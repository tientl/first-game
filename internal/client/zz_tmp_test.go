package client

import (
	"testing"

	"survivalgame/internal/sim"
)

func TestZZSplay(t *testing.T) {
	for _, k := range []sim.CreatureKind{sim.CreatureMammoth, sim.CreatureRabbit, sim.CreatureMonkey, sim.CreatureBear} {
		corpseCanvas(k, 1)
	}
}
