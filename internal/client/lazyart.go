package client

import (
	"sync"
	"sync/atomic"
	"time"

	"survivalgame/internal/sim"
)

// Art drawn when it is first wanted. The trees are sized by the player
// (several times their height), which makes them the biggest images of the
// world, so they are not drawn at start-up. Each set is drawn the first time
// it is wanted, on every core. The sets every world shows (the leafy trees
// and conifers, the prehistoric trees, the fruit trees) are drawn ahead on
// another goroutine as soon as the rest of the art is made (warmCommon); the
// rarer ones (the snow's pines, the swamp's trees, the woods) as soon as the
// snapshot holds one (prefetchArt), long before it comes into view, and a
// world without them never draws them.

// lazy is art drawn by gen the first time get is called.
type lazy[T any] struct {
	once    sync.Once
	done    atomic.Bool
	warming atomic.Bool
	v       T
	gen     func() T
}

// get draws the art if it is not drawn yet (or waits for it to be) and
// gives it.
func (l *lazy[T]) get() T {
	if l.done.Load() {
		return l.v
	}
	l.once.Do(func() {
		l.v = l.gen()
		l.done.Store(true)
	})
	return l.v
}

// warm draws the art on another goroutine, unless it is drawn or being
// drawn already.
func (l *lazy[T]) warm() {
	if l.done.Load() || !l.warming.CompareAndSwap(false, true) {
		return
	}
	go l.get()
}

// treeSet is the leafy trees of a size, the same hung with vines (at the
// same index), and the conifers.
type treeSet struct{ trees, vined, pines []sprite }

// initLazyTrees sets the generators of a's trees.
func (a *art) initLazyTrees() {
	for size := range 2 {
		s := treeScale[size]
		a.treeSets[size].gen = func() treeSet {
			var t treeSet
			t.trees, t.vined, t.pines = genTrees(s)
			return t
		}
		a.ancients[size].gen = func() ancientTrees { return genAncientTrees(s) }
		a.snowPineSets[size].gen = func() []sprite {
			p := make([]sprite, pineVariants)
			inParallel(pineVariants, func(v int) { p[v] = genPineOf(v, s, uint64(v)*13+1, true) })
			return p
		}
	}
	a.fruitSet.gen = genFruitArt
}

func (a *art) treesOf(size int) treeSet        { return a.treeSets[size].get() }
func (a *art) ancientOf(size int) ancientTrees { return a.ancients[size].get() }
func (a *art) snowPinesOf(size int) []sprite   { return a.snowPineSets[size].get() }
func (a *art) fruitOf() fruitArt               { return a.fruitSet.get() }

// warmCommon draws the trees every world shows on another goroutine, one set
// after another, each on every core.
func (a *art) warmCommon() {
	go func() {
		start := time.Now()
		for size := range 2 {
			a.treeSets[size].get()
		}
		for size := range 2 {
			a.ancients[size].get()
		}
		a.fruitSet.get()
		warmTime.Store(int64(time.Since(start)))
	}()
}

// warmTime is how long warmCommon took, once it is done (see ArtCost).
var warmTime atomic.Int64

// prefetchArt starts drawing the rarer trees as soon as the snapshot holds
// one, a screen or more before it is seen.
func (g *Game) prefetchArt() {
	if g.frame%20 != 0 || g.art == nil {
		return
	}
	for _, e := range g.curr.Entities {
		switch e.Kind {
		case sim.EntityTree:
			size := sizeIdx(e.Big)
			switch g.tileAt(int(e.TX), int(e.TY)) {
			case sim.TileSnow:
				g.art.snowPineSets[size].warm()
			case sim.TileMarsh:
				swampSets[size].warm()
			}
			if hasHive(e) {
				woodsSet.warm()
			}
		case sim.EntityGiantTree, sim.EntityBananaTree:
			woodsSet.warm()
		}
	}
	for _, c := range g.curr.Creatures {
		if c.Kind == sim.CreatureMonkey {
			woodsSet.warm()
		}
	}
}
