package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"

	"survivalgame/internal/sim"
)

// What a blow strikes decides how it looks. Sparks belong to stone and
// metal; a beast's hide, a spider's shell, a skeleton's bones and a tree's
// wood each throw their own things (the complaint was that hitting a
// dinosaur looked like hitting a lump of iron).

// hitMat is the stuff a struck thing is made of.
type hitMat uint8

const (
	matFlesh  hitMat = iota // hide, fur, scale, feather, skin: blood and a puff of its matter
	matChitin               // a shell: a crack, chips and a little ichor
	matBone                 // bones: a dry crack, chips and dust
	matStone                // rock and ore: sparks and grey chips
	matMetal                // anything metallic: sparks
	matWood                 // trees, wooden walls, shields: chips and splinters
	numHitMats
)

// hitMaterial is what creature k is made of; flesh unless it has a shell
// or is bone.
func hitMaterial(k sim.CreatureKind) hitMat {
	switch k {
	case sim.CreatureSpider, sim.CreatureScorpion, sim.CreatureCrab:
		return matChitin
	}
	return matFlesh
}

// entityHitMat is what a tool strikes in entity kind k.
func entityHitMat(k sim.EntityKind) (hitMat, bool) {
	switch k {
	case sim.EntityTree, sim.EntityFigTree, sim.EntityGinkgoTree, sim.EntityPalm, sim.EntityBananaTree,
		sim.EntityWall, sim.EntityCactus:
		return matWood, true
	case sim.EntityBoulder, sim.EntityCoal, sim.EntityStoneWall:
		return matStone, true
	}
	if isOre(k) {
		return matMetal, true
	}
	return matFlesh, false
}

// What a beast sheds when struck: fur tufts, scale flecks, feathers, or
// nothing but blood.
type matterKind uint8

const (
	mNone matterKind = iota
	mFur
	mScale
	mFeather
	mShell // chitin
	mBone
	mWood
)

type matterDef struct {
	kind matterKind
	cols [3]color.RGBA // dark, mid, light
}

const (
	mtSkin = iota
	mtFurBrown
	mtFurGrey
	mtFurTan
	mtScaleGreen
	mtScaleOlive
	mtScaleTan
	mtScaleBlue
	mtFeatherDark
	mtFeatherRust
	mtShellDark
	mtShellTan
	mtShellRed
	mtBone
	mtWood
	mtBark
)

var matters = [...]matterDef{
	mtSkin:        {mNone, [3]color.RGBA{hex(0x6a4030), hex(0xa06a4a), hex(0xd49a76)}},
	mtFurBrown:    {mFur, [3]color.RGBA{hex(0x3a2418), hex(0x6a4428), hex(0x96683c)}},
	mtFurGrey:     {mFur, [3]color.RGBA{hex(0x6a6862), hex(0xa09c92), hex(0xd8d4c8)}},
	mtFurTan:      {mFur, [3]color.RGBA{hex(0x7a5a2c), hex(0xb08a46), hex(0xdcb872)}},
	mtScaleGreen:  {mScale, [3]color.RGBA{hex(0x2c4a2c), hex(0x4a7a3c), hex(0x8cb460)}},
	mtScaleOlive:  {mScale, [3]color.RGBA{hex(0x3a3a22), hex(0x6a6a38), hex(0xa0a064)}},
	mtScaleTan:    {mScale, [3]color.RGBA{hex(0x5a4a2a), hex(0x8e7a48), hex(0xc4b078)}},
	mtScaleBlue:   {mScale, [3]color.RGBA{hex(0x1e3a4a), hex(0x3a6a80), hex(0x9cc8d6)}},
	mtFeatherDark: {mFeather, [3]color.RGBA{hex(0x2a2428), hex(0x544a52), hex(0x948a90)}},
	mtFeatherRust: {mFeather, [3]color.RGBA{hex(0x5a2e1c), hex(0x9a5a30), hex(0xdca468)}},
	mtShellDark:   {mShell, [3]color.RGBA{hex(0x16120e), hex(0x3a2e24), hex(0x7a6a58)}},
	mtShellTan:    {mShell, [3]color.RGBA{hex(0x5a3a1c), hex(0x8c6232), hex(0xc29a58)}},
	mtShellRed:    {mShell, [3]color.RGBA{hex(0x7a2214), hex(0xc4482a), hex(0xe88a64)}},
	mtBone:        {mBone, [3]color.RGBA{hex(0x8a7e62), hex(0xcdc2a4), hex(0xf0e8d0)}},
	mtWood:        {mWood, [3]color.RGBA{hex(0x5e3c20), hex(0x8a5c32), hex(0xc99a5e)}},
	mtBark:        {mWood, [3]color.RGBA{hex(0x2e1e12), hex(0x4e3420), hex(0x7a5636)}},
}

// hitMatter is the matter creature k sheds (an index of matters).
func hitMatter(k sim.CreatureKind) int {
	switch k {
	case sim.CreatureBear, sim.CreatureMammoth, sim.CreatureMammothKing, sim.CreatureMonkey:
		return mtFurBrown
	case sim.CreatureRabbit:
		return mtFurGrey
	case sim.CreatureSabre:
		return mtFurTan
	case sim.CreatureTrike, sim.CreatureToro, sim.CreatureStego, sim.CreatureBrachio, sim.CreatureThesc,
		sim.CreatureAnkylo, sim.CreatureToad, sim.CreatureGlowToad, sim.CreatureWorm:
		return mtScaleGreen
	case sim.CreatureRex, sim.CreatureSpino, sim.CreatureCarcharo, sim.CreatureGiganoto, sim.CreatureCroc,
		sim.CreatureBrachy, sim.CreatureChampso, sim.CreatureTitanoboa:
		return mtScaleOlive
	case sim.CreatureSnake, sim.CreaturePtero, sim.CreatureQuetzal:
		return mtScaleTan
	case sim.CreatureFish, sim.CreaturePlesio, sim.CreatureMosa, sim.CreatureKraken, sim.CreatureMegalodon:
		return mtScaleBlue
	case sim.CreatureTroodon:
		return mtFeatherDark
	case sim.CreatureRaptor, sim.CreatureAchero, sim.CreatureAnzu, sim.CreatureOrnitho:
		return mtFeatherRust
	case sim.CreatureSpider:
		return mtShellDark
	case sim.CreatureScorpion:
		return mtShellTan
	case sim.CreatureCrab:
		return mtShellRed
	}
	return mtSkin
}

var (
	palBlood = hexes(0x24060a, 0x4e0a12, 0x7e1220, 0xa81c2c)
	palIchor = hexes(0x4e6a14, 0x86a626, 0xc4d85c)
)

// --- what the blow's drops are -----------------------------------------------------

type hitDrop struct {
	vx, vy float64 // art pixels a second
	off    float64 // where it lands, below the foot line
	size   float64
}

// hitDrops are the drops of a blow: n of them thrown mostly along ang. The
// same seed gives the same drops, so the splat they leave (fxSplat) knows
// where they land.
func hitDrops(r *fxRng, n int, ang, k float64) []hitDrop {
	out := make([]hitDrop, n)
	for i := range out {
		a := ang + r.in(-0.85, 0.85) - 0.25
		if i%4 == 3 {
			a = ang + math.Pi + r.in(-0.8, 0.8) // a few thrown back
		}
		v := r.in(70, 190) * k
		out[i] = hitDrop{math.Cos(a) * v, math.Sin(a)*v*0.7 - r.in(40, 90), r.in(-2, 3), r.in(0.8, 1.5)}
	}
	return out
}

const dropGrav, dropDrag = 520.0, 1.1

// land is when drop d, thrown from height z above the foot line, lands, and
// where (x offset); ok is false if it does not within a second.
func (d hitDrop) land(z float64) (t, dx float64, ok bool) {
	for t = 0.02; t < 1; t += 1.0 / 120 {
		px, py, _, vy := ballistic(0, -z, d.vx, d.vy, dropGrav, dropDrag, t)
		if py >= d.off && vy > 0 {
			return t, px, true
		}
	}
	return 0, 0, false
}

// --- the effects ---------------------------------------------------------------------

// hitFX: a blow landing on mat. lvl is 0 for an everyday blow, 1 for a heavy
// one, 2 for a critical; matter is an index of matters; ground the dust of
// the ground.
func hitFX(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, lvl int, mat hitMat, matter, ground int, t float64) {
	switch mat {
	case matFlesh:
		fxHitFlesh(b, r, x, y, gy, ang, k, lvl, matter, ground, t)
	case matChitin:
		fxHitChitin(b, r, x, y, gy, ang, k, lvl, matter, t)
	case matBone:
		fxHitBone(b, r, x, y, gy, ang, k, lvl, t)
	case matWood:
		fxHitWood(b, r, x, y, gy, ang, k, matter, t)
	}
}

// creamFlash is the brief pale flash of a blow: a squashed ring and a
// crescent across the way it went, in two or three frames.
func creamFlash(b *fxBatch, x, y, ang, k float64, lvl int, t float64) {
	const dur = 0.055
	if t >= dur+0.03 {
		return
	}
	u := clamp01(t / dur)
	big := 1 + 0.35*float64(lvl)
	if t < dur {
		b.glow(x, y, 13*k*big, palCream[1], 0.28*(1-u))
		rad := (3.5 + 8*outCubic(u)) * k * big
		b.ring(x, y, rad-1.4*k*(1-u)-0.4, rad, 0.55, 0, 2*math.Pi, palCream[2], 0.7*(1-u), 0.25)
		cs, sn := math.Cos(ang), math.Sin(ang)
		cr := rad * 1.15
		b.ring(x+cs*2.5*k, y+sn*2.5*k, cr-2.2*k*(1-u)-0.4, cr, 1, ang-1.0, ang+1.0, palCream[3], 1-u*0.8, 0.2)
		b.diamond(x, y, (2.2+1.2*float64(lvl))*k*(1-u*0.6), palCream[3], 1-u, 0)
	}
}

func fxHitFlesh(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, lvl, matter, ground int, t float64) {
	z := gy - y
	creamFlash(b, x, y, ang, k, lvl, t)
	if lvl >= 2 { // a critical: pale rays, no fire
		if t < 0.09 {
			u := t / 0.09
			for i := 0; i < 8; i++ {
				a := float64(i)*math.Pi/4 + ang*0.3
				l := (9 + 8*float64(i%2)) * k * (0.5 + outCubic(u)*0.8)
				b.taper(x+math.Cos(a)*3, y+math.Sin(a)*3*0.8, x+math.Cos(a)*l, y+math.Sin(a)*l*0.8, 1.6*k*(1-u), 0.2, palCream[2], 1-u, 0.2)
			}
		}
	}
	// A smear of blood along the blow for the heavy ones.
	if lvl >= 1 && t < 0.2 && t > 0.015 {
		u := (t - 0.015) / 0.185
		l := (9 + 6*float64(lvl)) * k * outCubic(u*1.5)
		b.taper(x, y, x+math.Cos(ang)*l, y+math.Sin(ang)*l*0.8, 2.6*k*(1-u), 0.3, palBlood[2], 0.85*(1-u), 0)
	}
	n := 4 + 3*lvl + int(k*1.5)
	for _, d := range hitDrops(r, n, ang, k) {
		lt, _, ok := d.land(z)
		if !ok || t >= lt {
			continue
		}
		px, py, vx, vy := ballistic(x, y, d.vx, d.vy, dropGrav, dropDrag, t)
		c := palBlood[1+int(d.size*2)%3]
		b.streak(px, py, vx, vy, 0.016, d.size*1.15, c, 1, 0)
		b.dot(px, py, d.size*0.9, palBlood[3], 0.9, 0)
	}
	fxMatterPuff(b, r, x, y, gy, ang, k, lvl, matter, ground, t)
}

// fxMatterPuff: what the struck beast sheds: tufts of fur, flecks of scale,
// feathers, or only the dust it was standing in.
func fxMatterPuff(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, lvl, matter, ground int, t float64) {
	m := matters[min(max(matter, 0), len(matters)-1)]
	n := 3 + lvl*2
	switch m.kind {
	case mFur:
		for i := 0; i < n+2; i++ {
			a := ang + r.in(-1.3, 1.3)
			v := r.in(25, 75) * k
			life := r.in(0.5, 0.75)
			ph := r.in(0, 6)
			if t >= life {
				continue
			}
			px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-12, 70, 3, t)
			u := t / life
			al := 1 - u*u
			sw := math.Sin(t*14+ph) * 1.2
			c := m.cols[i%3]
			b.taper(px, py, px-vx*0.05+sw, py-vy*0.05-1.5, 1.5*k, 0.3, c, al, 0)
			b.taper(px, py, px-vx*0.04-sw, py-vy*0.05+1.2, 1.1*k, 0.2, m.cols[(i+1)%3], al, 0)
		}
	case mScale:
		for i := 0; i < n; i++ {
			a := ang + r.in(-1.2, 1.2) - 0.3
			v := r.in(50, 130) * k
			life := r.in(0.3, 0.5)
			if t >= life {
				continue
			}
			px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-40, 420, 1.5, t)
			th := t*18 + float64(i)
			cs, sn := math.Cos(th)*1.0*k, math.Sin(th)*1.0*k
			c := m.cols[1+i%2]
			b.quad(px-cs, py-sn*0.6, px-sn*0.4, py+cs*0.5, px+cs, py+sn*0.6, px+sn*0.4, py-cs*0.5, c, 1-t/life*0.5, 0)
		}
		if ground <= dustGrass {
			puff(b, r, x+math.Cos(ang)*3, gy, math.Cos(ang)*14, t, 0.5, 0.6*k, dustPals[ground])
		}
	case mFeather:
		for i := 0; i < n+1; i++ {
			a := ang + r.in(-1.4, 1.4) - 0.2
			v := r.in(30, 90) * k
			life := r.in(0.6, 0.95)
			ph := r.in(0, 6)
			if t >= life {
				continue
			}
			px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-20, 45, 2.6, t)
			px += math.Sin(t*9+ph) * 2.5
			u := t / life
			al := 1 - u*u
			rot := math.Sin(t*7+ph)*0.9 + 1.2
			l := 3.2 * k
			cs, sn := math.Cos(rot)*l, math.Sin(rot)*l
			b.taper(px-cs, py-sn, px+cs, py+sn, 0.2, 1.6*k, m.cols[1+i%2], al, 0)
			b.taper(px-cs*0.9, py-sn*0.9, px+cs, py+sn, 0.1, 0.5, m.cols[0], al*0.8, 0)
		}
	default:
		if ground <= dustGrass {
			puff(b, r, x+math.Cos(ang)*3, gy, math.Cos(ang)*14, t, 0.45, 0.5*k, dustPals[ground])
		}
	}
}

// splatFX: the little splats the drops of fxHit leave where they land, which
// fade in a second or two. It shares fxHit's seed. The ground effect.
func splatFX(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, lvl int, t float64) {
	z := gy - y
	n := 4 + 3*lvl + int(k*1.5)
	for _, d := range hitDrops(r, n, ang, k) {
		lt, dx, ok := d.land(z)
		if !ok || t < lt {
			continue
		}
		age := t - lt
		al := clamp01((1.5 - age) / 0.9)
		al *= math.Min(1, age/0.03+0.4)
		w := (1.0 + d.size*0.9) * math.Min(1, 0.5+age*8)
		b.disc(x+dx, gy+d.off, w, w*0.5, palBlood[1], 0.8*al, 0)
		b.disc(x+dx-w*0.15, gy+d.off-0.2, w*0.55, w*0.28, palBlood[2], 0.7*al, 0)
	}
}

func fxHitChitin(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, lvl, matter int, t float64) {
	m := matters[min(max(matter, 0), len(matters)-1)]
	// A pale crack: thin forked lines.
	var ca, cl [4]float64
	for i := range ca {
		ca[i] = ang + math.Pi*float64(i)/2 + r.in(-0.35, 0.35)
		cl[i] = r.in(6, 12)
	}
	if t < 0.1 {
		u := t / 0.1
		for i := 0; i < 4; i++ {
			a := ca[i]
			l := cl[i] * k * (1 + 0.3*float64(lvl)) * (0.5 + outCubic(u*1.6)*0.5)
			cs, sn := math.Cos(a), math.Sin(a)
			mx, my := x+cs*l*0.55, y+sn*l*0.55*0.85
			ex, ey := mx+math.Cos(a+0.5)*l*0.5, my+math.Sin(a+0.5)*l*0.5*0.85
			b.taper(x, y, mx, my, 1.1*k, 0.7, palCream[3], 1-u, 0.15)
			b.taper(mx, my, ex, ey, 0.7, 0.2, palCream[2], 1-u, 0.1)
		}
		b.glow(x, y, 9*k, palCream[2], 0.3*(1-u))
	}
	for i := 0; i < 6+2*lvl; i++ {
		a := ang + r.in(-0.9, 0.9) - 0.2
		v := r.in(60, 160) * k
		life := r.in(0.3, 0.5)
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-45, 480, 1.4, t)
		py = math.Min(py, gy)
		th := t*16 + float64(i)
		cs, sn := math.Cos(th)*1.1*k, math.Sin(th)*1.1*k
		b.quad(px-cs, py-sn*0.6, px-sn*0.4, py+cs*0.5, px+cs, py+sn*0.6, px+sn*0.4, py-cs*0.5, m.cols[i%3], 1-t/life*0.4, 0)
	}
	for i := 0; i < 3+lvl; i++ { // ichor
		a := ang + r.in(-0.7, 0.7)
		v := r.in(50, 120) * k
		life := r.in(0.3, 0.45)
		if t >= life {
			continue
		}
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-30, 460, 1.2, t)
		if py > gy {
			continue
		}
		b.streak(px, py, vx, vy, 0.014, 1.1, palIchor[1+i%2], 1, 0)
	}
}

func fxHitBone(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, lvl int, t float64) {
	m := matters[mtBone]
	if t < 0.08 { // a dry crack: a thin cross and a small ring
		u := t / 0.08
		l := (7 + 3*float64(lvl)) * k * (0.6 + 0.4*outCubic(u))
		for _, d := range [2]float64{0.7, -0.7} {
			cs, sn := math.Cos(ang+d), math.Sin(ang+d)
			b.taper(x-cs*l, y-sn*l*0.85, x+cs*l, y+sn*l*0.85, 0.2, 0.2, palCream[3], 1-u, 0.2)
			b.taper(x-cs*l*0.5, y-sn*l*0.43, x+cs*l*0.5, y+sn*l*0.43, 1.2*k*(1-u), 1.2*k*(1-u), palCream[3], 1-u, 0.2)
		}
		rad := (3 + 6*outCubic(u)) * k
		b.ring(x, y, rad-0.8, rad, 0.7, 0, 2*math.Pi, palCream[2], 0.8*(1-u), 0.2)
	}
	for i := 0; i < 6+2*lvl; i++ {
		a := ang + r.in(-1.0, 1.0) - 0.2
		v := r.in(60, 150) * k
		life := r.in(0.3, 0.5)
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-45, 480, 1.4, t)
		py = math.Min(py, gy)
		th := t*14 + float64(i)
		l := (1.2 + 0.5*float64(i%3)) * k
		cs, sn := math.Cos(th)*l, math.Sin(th)*l
		b.taper(px-cs, py-sn*0.7, px+cs, py+sn*0.7, 1.5*k, 1.0*k, m.cols[1+i%2], 1-t/life*0.4, 0)
	}
	puff(b, r, x, gy, math.Cos(ang)*14, t, 0.55, 0.6*k, [3]color.RGBA{hex(0x6a665c), hex(0x9a958a), hex(0xd0cbbd)})
}

// fxHitWood: chopping: chips and splinters flung back off the cut, bark, and
// a small pale flash. ang is the way the blow went into the wood.
func fxHitWood(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, matter int, t float64) {
	if t < 0.05 {
		u := t / 0.05
		b.glow(x, y, 7*k, palCream[1], 0.22*(1-u))
		rad := (2.5 + 5*outCubic(u)) * k
		b.ring(x, y, rad-1.0*k*(1-u)-0.3, rad, 0.6, 0, 2*math.Pi, palCream[2], 0.6*(1-u), 0.2)
	}
	pal := matters[mtWood].cols
	bark := matters[mtBark].cols
	for i := 0; i < 9; i++ {
		a := ang + math.Pi + r.in(-1.1, 1.1)
		v := r.in(50, 140) * k
		life := r.in(0.35, 0.6)
		spin := r.in(10, 24)
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-r.in(30, 70), 460, 1.0, t)
		py = math.Min(py, gy)
		th := t*spin + float64(i)
		l := (1.3 + 1.6*float64(i%3)/2) * k
		cs, sn := math.Cos(th)*l, math.Sin(th)*l
		c := pal[i%3]
		if i%3 == 2 {
			c = bark[1+i%2]
		}
		if i < 5 { // splinters: long and thin
			b.taper(px-cs, py-sn*0.7, px+cs, py+sn*0.7, 0.7*k, 0.2, c, 1-t/life*0.4, 0)
		} else {
			b.quad(px-cs*0.6, py-sn*0.4, px-sn*0.5, py+cs*0.4, px+cs*0.6, py+sn*0.4, px+sn*0.5, py-cs*0.4, c, 1-t/life*0.4, 0)
		}
	}
}

// spawnHit starts a blow landing on mat at (x, y), z up: its effect, and for
// flesh the splats its drops will leave.
func (g *Game) spawnHit(x, y, z, ang, k float64, lvl int, mat hitMat, matter int) {
	pal := g.dustAt(x, y)
	seed := (g.fx.seq + 1) * 2246822519
	f := fxInst{kind: fxHit, x: x, y: y, z: z, ang: ang, k: k, lvl: lvl, mat: mat, matter: matter, pal: pal, seed: seed, keep: true}
	g.spawnFX(f)
	if mat == matFlesh {
		f.kind = fxSplat
		g.spawnFX(f)
	}
}

// toolHitFX is a tool landing on entity e at (x, y), the blow going ang:
// chips off wood, sparks and grey chips off rock and ore. Hands and hoes on
// plants throw nothing.
func (g *Game) toolHitFX(e protocol.EntityView, x, y, ang float64, how swingKind) {
	mat, ok := entityHitMat(e.Kind)
	if !ok || how == swingHand || how == swingHoe {
		return
	}
	if mat == matWood && sim.IsFruitTree(e.Kind) && e.Ready {
		return // ripe fruit is picked, not chopped
	}
	switch mat {
	case matWood:
		z := 14.0
		if e.Kind == sim.EntityWall || e.Kind == sim.EntityCactus {
			z = 10
		}
		g.spawnFX(fxInst{kind: fxHit, x: x, y: y + 0.2, z: z, ang: ang, k: 1, mat: matWood})
		if g.leafyAt(x, y) && e.Kind != sim.EntityWall {
			g.spawnFX(fxInst{kind: fxLeaves, x: x, y: y, z: 30, k: 1})
		}
	default: // rock and ore: sparks, grey chips and dust
		z := 8.0
		g.spawnFX(fxInst{kind: fxSpark, x: x, y: y + 0.2, z: z, ang: ang + math.Pi, k: 0.7})
		g.spawnFX(fxInst{kind: fxBits, x: x, y: y + 0.2, z: z, ang: ang + math.Pi, k: 1, pal: g.dustAt(x, y)})
	}
}
