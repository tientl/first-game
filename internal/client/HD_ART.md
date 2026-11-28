# HD art: how to redraw a sprite at 3x

The world is laid out in **art pixels** (a tile is `TileW` x `TileH` = 24 x 18)
and drawn on a canvas as big as the screen, **R** screen pixels to an art pixel
(R = 2 at 720p, 3 at 1080p, plus the zoom). Redrawn art is made at
`HDDensity` = 3 image pixels per art pixel, so at 1080p it lands one image
pixel on one screen pixel. Old art (density 1) still draws exactly as before,
so art can be swapped one sprite at a time. The engine side is `hd.go`.

## Coordinates: everything stays in art pixels

- Positions, offsets, anchors, `TileW`/`TileH`, shadow widths, label heights:
  all in art pixels, as today. Never multiply by 3 in drawing code.
- A `sprite` is `{img, ax, ay, dens}`. **`ax, ay` are in art pixels** for
  every density (image pixel `(3*ax, 3*ay)` for an HD sprite).
- An HD sprite covers the same art area as the 1x one it replaces: a 20 x 16
  art-pixel bush is a 60 x 48 image.
- Size a sprite with `s.w()`, `s.h()` (art pixels), **not**
  `s.img.Bounds()` (image pixels). Grep the code that uses your sprite for
  `.img.Bounds()` and `.img.SubImage` before switching it.

## Making an HD sprite

Draw in code on a canvas 3x finer; the shape calls take design units (art
pixels), so the same code can make both densities (see `hdtest.go`):

```go
c := newScaledCanvas(26, 20, HDDensity) // 26 x 20 art pixels, 78 x 60 image pixels
c.ellipse(13, 13, 11.5, 7, ...)          // design units = art pixels
// per-pixel callbacks get image pixels; c.lx(x), c.ly(y) give art units
c.outline(0.8)                           // a 1-image-pixel outline: a third as thick
s := mkHD(c, 13, 19)                     // anchor in art pixels
```

From a PNG drawn at 3x: `spriteHD(img, ax, ay)` with `ax, ay` in art pixels.

Keep a 1x fallback while trying it out: keep the old `mk(...)` sprite next to
it and pick one with a flag, or make both from the same code with
`newScaledCanvas(w, h, 1)` and `newScaledCanvas(w, h, HDDensity)`.

## Drawing

- `drawAt`, `drawSwaying` and `drawSprite(dst, s, op)` handle any density.
  Write `op` as for a 1x sprite (`op.GeoM.Translate(-s.ax, -s.ay)` ...);
  `drawSprite` folds the density in.
- `dst.DrawImage(s.img, op)` on a sprite is a bug once `s` is HD (it would
  be drawn 3x too big): use `drawSprite`.
- World drawing functions take `dst drawTarget` (the world pass, or a plain
  image such as the screen or a scratch buffer). Vector shapes go through
  `fillRect`, `strokeLine`, `fillCircle`, `strokeCircle`, `strokeRect`;
  colour matrices through `drawColorM`.
- HD images are drawn straight onto the full-resolution canvas: nearest
  filtering when they land 1:1 (1080p) or at a whole scale, linear when not
  (720p: 2/3). Density-1 art goes to an art-pixel layer that is copied up R
  times bigger, so it keeps its pixel grain.
- Moving things are drawn on their own sub-pixel grid: use
  `cam.toScreenFine` for a lerped position and call `align(dst, sx, sy)`
  before drawing it (`drawSprites` does this for everything in the sorted
  pass). Static things use `cam.toScreen`. Place parts of a figure at whole
  art pixels from its point (`sx + math.Floor(dx)`, not
  `math.Floor(sx + dx)`).
- Shaders (`DrawRectShader`) and triangles on the world still run at
  art-pixel resolution. If HD water or lighting needs full resolution, add a
  `drawFine`-style path in `hd.go` (flush, then draw on `wd.canvas` scaled by
  `wd.r`).
- The interface draws on a `uiDst`, the screen in UI pixels: `drawSprite`
  sizes HD sprites right there too. See "The interface" and "Item icons"
  below.
- A figure drawn through a scratch image (struck in `jolt.go`, cut off at
  the water in `searender.go`) uses a `fineBuf` (`hd.go`): a drawTarget in
  art pixels on an image HDDensity times finer, put on the world with
  `drawImageD(dst, buf.img, op, HDDensity)`.
- The player is HD (`lookhd.go`, `lookhd_head.go`, `armorhd.go`, drawn with
  the shapes and shaders of `hdfig.go`); `playerHD = false` brings back the
  density-1 figure of `look.go`.
- The player has joints (`limbs.go`, `limbdraw.go`): arms and legs are
  segments about a shoulder, an elbow, a hip, a knee and an ankle, posed in
  the figure's own space and projected for each facing. A frame is two
  halves put together when drawn: the legs (`loKey`: a walk or a run of 16
  frames, matched to the ground covered so a foot that is down stays put)
  and everything above them (`upKey`: the arms through a stride, or the six
  frames of a blow, the tool gripped at the hand). Each half is drawn the
  first time it is shown. `playerJointed = false` brings back the stiff
  eight-frame figure. `LIMB_ANIM=dir go test ./internal/client -run
  DumpLimbAnim -v` writes sheets and GIFs of the strides and blows and what
  they cost.
- The dinosaurs drawn from PNGs are HD: `dinos/hd/*.png` are the drawings
  of `dinos/` redrawn 3x as fine. The rex and the three-horn are painted
  over their drawings by `go run ./cmd/dinopaint` from descriptions in
  `internal/dinopaint` (masses, muscles, materials, scales, eyes, teeth,
  horns; see `dinos/hd/README.md` to add one); the others are still upscaled
  by `dinos/hd/gen.py`. An open jaw at HD shows a mouth with a tongue and
  lower teeth (`mouthHD`); a rig whose drawing has the upper teeth above
  its jaw line says `drawnTeeth`. The rig (`dinorig.go`) is still
  measured in the pixels of the density-1 drawing and scales itself; its
  canvases carry `dens`, which `mk` turns into an HD sprite. `dinoHD = false`
  brings back the density-1 drawings. Strides, wingbeats and wound-up blows
  are extra rig poses drawn the first time they are shown (`dinoanim.go`);
  `DINO_ANIM=dir go test ./internal/client -run DumpDinoAnim -v` writes
  their frames and what they cost.
- The animals drawn in code are HD (`critterhd.go` holds the tools and the
  frames; `critsmall.go`, `critcroc.go`, `critbeast.go`, `critcave.go` the
  drawings): bodies of blended signed-distance shapes in art pixels, legs of
  two bones that reach for where the gait puts each foot, so a walk is
  `cycFrames` frames and a blow is wound up over `blowFrames` before it
  lands. `critter(kind, side, pose, f)` draws a frame the first time it is
  shown; frame 0 of a pose is the pose the density-1 art had. `critterHD =
  false` (`shot -critterhd=false`) brings the old drawings back.
  `CRITTER_DUMP=dir go test ./internal/client -run DumpCritters -v` writes a
  sheet (old above new, then the moving frames) and GIFs of each animal, and
  what drawing them costs.
  Most of them are painted as the dinosaurs are (`critpaint.go`): a part
  drawn with `q.body(shape, surf, look)` keeps, per pixel, the way its
  surface faces, its material (`surf`: fur in locks, scales, shell, wet
  skin, plates, matte hide, each on a `richRamp` leaning violet in the
  shadow and warm in the light) and how much it is shadowed; a part laid
  over another gets a crease and a soft shadow down and to the right. When
  the drawing is done (`q.done`) every such pixel is lit at once from the
  top left, with a rim of light on the edges turned away and a glint on what
  shines, and outlined in the part's own darkest tone, lighter on the lit
  side. `q.coat` repaints part of a form in another material (a pale belly,
  war paint, a dark finger tip) and `q.shade` darkens or lightens it (a
  groove, a brow), both keeping its form; eyes, teeth and glints go on flat
  (`q.flat`, `q.dot`, `q.glassEye`). The files: `critsmall.go` (rabbit,
  worm, fish, snake), `critcrab.go`, `critmonkey.go`, `critspider.go`,
  `critscorp.go`, `critcroc.go`, `crittribe.go` (the warrior, drawn as tall
  and as long in the leg as the player), and `crittotem.go`, the village
  post, painted the same way and made a sprite with `mkUp`. A smaller
  animal is drawn smaller with `newCritZoom` (see "Sizes" below).
- Everyone starts as a caveman: the jointed figure bare (`cavefig.go`), a
  man in a fur loincloth, a woman in a fur wrap and skirt, both barefoot.
  The clothes (`sim.Wear`, made and carried like armour) go over it: the
  tunic and the mantle in `tallfig.go`, leg wraps and boots in `limbs.go`.
  A bare arm is a limb whose reach from its axis is given down its length
  on each side (`prof`, `sLimb`), lit softly, with each muscle (the cap of
  the shoulder, biceps, triceps, the forearm's masses) lit as a bulge of
  its own on it. The head (`tallhead.go`) has small eyes under the brow and
  a man an untrimmed beard (`beardShade`); with the long braid of hair his
  beard is plaited at the chin. Seen from the front or the back a stride is
  drawn shallower than the ground it covers (`legK`), so the feet stay on
  the line the figure stands on. `CAVE_DUMP=dir go test ./internal/client
  -run DumpCave` writes sheets of both figures, each of the clothes, the
  poses, and GIFs of the walk and the run from the side, the front and the
  back.
- The fallen body and its bones (`bodyart.go`) are the jointed figure laid
  down, in what it wore; the map's head marker is its head. Not
  density-aware yet: the sprite export in `export.go`.

## Motion reference (LPC)

The player's motions (`motion.go`) follow the animation set of the
Universal LPC sheets: their key poses and their rhythm, drawn on our own
figure. No LPC art is used or kept in the repository (most of it is
CC-BY-SA or GPL with a no-DRM clause); the sheets were only studied, and
`motion_compare.gif` (made outside the repository) sets theirs beside ours
for review. LPC frames are 64 x 64, 13 to a row, rows facing up, left,
down, right; its body is about 47 pixels tall where ours is 93, so one of
their pixels is two of ours. Its previewer steps every animation at 8
frames a second (125 ms) and holds a pose by repeating it in the cycle.

What each LPC animation does, measured off the body and head layers:

| LPC | frames, time | what happens | ours |
|---|---|---|---|
| walk | 8 + stand, 1 s | dips 1 px on the two contact frames, level between; arms swing against the legs | `stepGait`: dips 2 px at contact (`bounce -1`), arm swing 0.5 rad; the stride stays matched to the ground covered |
| run | 8, 1 s | sinks 1 px as a foot takes the weight, rises 2 in the air; arms bent square, trunk forward | `runGait` already did this (5-6 px), kept |
| idle | 2 poses, 0 0 1 (375 ms) | the second rises 1 px: a breath | `moIdle`: the trunk fills 1.5 px, arms float out; slowed to 2.3 s, as a 375 ms twitch reads as a jitter at our size |
| combat idle | 2 poses, 0 0 1 | knees bent, weapon up in front, a 2 px dip on the off beat | `moReady`: 250/125 ms, dip 2.5 px, held after a blow or with a beast on the attack within 6 tiles |
| slash | 6, 750 ms | ready; a 1-2 px dip with the weapon cocked low behind (anticipation); behind; the sweep (contact); rising (follow-through); held | `moChop` (axe, pick, hoe) and `moSlash` (a sword's forehand): the same beats over the tool's angle; the sim's timing (`chopAngle`, `slashAngle`, `cleaveAngle`) is unchanged, the frame is picked from the angle |
| 1h backslash | 0-5 then 7-12, 1.5 s | a forehand from over the head down across, then a backhand low to high; the head dips 1-3 px on the strikes | `moBackslash`: a sword's blows alternate forehand and backhand |
| thrust | 8, 1 s | ready; drawn back with the weight back (2 frames); driven out (contact); held 3 frames with a 1 px dip; back | `moThrust`, with a lunge, on the spear's 12-frame clock (`thrustFrame`) |
| shoot | 13, 1.6 s | bow up (3), nock, draw to the cheek (2), hold (3), loose, down (3) | `moShoot`, ready for a bow (none in the game yet; the sheet draws one in code); `moThrow` is it short and underhand, for a toss |
| hurt | 6, 750 ms, facing down only | head top 15, 16, 23, 27, 33, 43: knees go, a kneel, a slump, face down | its timing for the flinch (`moHurt`, over the jolt's 14 frames); its poses for the fall (`moDie`), ending on the body as it lies |
| sit | 3 poses, 625 ms each | down on the ground, the head 8-10 px lower | `moSitDown`, `moSit`, `moStandUp`: by a burning fire after 4 s standing still; up at once on moving, at twice the speed |
| emote, spellcast | 3 poses held 625 ms; 7 in 875 ms | hands to the chest, arms up; hands rise and spread, held, down | the rhythm of eating and drinking (`moEatBite`, `moEatHand`, `moEatBowl`, `moDrink`): up, held at the mouth, down |
| jump | 5, ~750 ms | a 3 px crouch, launch, 3-4 px up, land | its rhythm for getting on and off a mount; no jump in the game |
| climb, halfslash | | | not used: nothing climbs; no weapon needs a third blow |

What LPC has no row for is ours, timed the same way: the crouch to pick
something up and the reach into a tree (`moGather`, `moReachUp`), scooping
at the water on one knee (`moScoop`; cupped hands `moDrinkWater`, ready for
a drink from the water the game does not have yet), the breaststroke and
treading water (`moSwim` by the water covered, `moTread`; `swimhd.go`),
sleeping (`moSleep`, laid down under a fur in `sleephd.go`; `moWake`), and
riding (`moRide`, `moRideRun`, `moRideWide`, `moRideIdle`, `moMount`,
`moDismount`; `ridehd.go`).

How a motion is drawn: a key pose (`kp`) sets both halves of the figure;
the trunk's lean, the head's nod and the jaw are drawn bent, not skewed:
every shape, dot and shader of the figure goes through `figHD.xf`
(`hdfig.go`), so a lean is drawn again, crisp. Which motion a player is in
is read off the snapshots (`motionpick.go`): the deed counter the server
sends (`sim/deeds.go`: eating, scooping, tossing), a blow begun, a mount
gained or lost, death, sleep; and off the client's own clock.

Layers, back to front, of a standing figure (`drawFig`): what the hand
holds when facing up; the far sleeve (`upFar`); the far hand and the hair
behind the body (`upBack`); the legs; the bare trunk and the fur (`upBody`);
the clothes (`upShirt`); the body armour; the sleeves (`upArms`); the head,
the hands, the belt and the mantle (`upTop`); the helmet; what is held,
facing down or sideways; a raised arm (`upArmsOver`) and its fist
(`upTopOver`). Sideways the far arm is drawn behind the trunk; from behind
the mantle covers the back and long hair lies over the mantle (the head is
laid over the body's layers); a hand up at the mouth comes over the face,
or from behind goes out of sight behind the head (`kp.fore0`). A rider's
far leg is drawn before the mount, the rest after; a swimmer under the
surface is drawn faint before what is above it.

Colours come from tables, not code: skin from `skinTones`, hair from
`hairColors` (`look.go`), and the clothes are drawn in greys and tinted
with the player's colour as they are drawn (`upperTinted`); a new skin,
hair or cloth colour is a new entry.

A figure turned whole (lying back onto a bed, going over in a fall) is
not turned at draw: its pose sets `kp.spin`, and both halves and the head
are drawn through the turn (the hips dropped first), so it comes out
crisp; such a frame is put together on the CPU (`standingCanvas`) and
kept as one sprite (`laidSprite`). The fall ends on the body's own drawing
(`graveSprite`), on the tile the server laid it, the figure sliding there
as it goes, so the body takes over without a jump; the body stays a tile
entity and the server does not change. Setting off from sitting is
`moRise`, the walk rising out of a crouch over 0.55 s, so the controls
answer at once. Loose hair (`hairfloat.go`) lies on the pillow and floats
behind a swimmer for every style.

What it costs: every frame of every motion of one look is 2216
half-frames, 19 MB if all were shown, 1.3 ms each to draw; they are drawn
the first time they are shown, so a session pays for what it shows.
`go run ./cmd/shot -w 1920 -h 1080 -motionsheet dir` plays every motion
through the game's own drawing and writes `motion_sheet.png` and a GIF of
each group; `MOTION_POSES=dir go test ./internal/client -run
DumpMotionPoses` writes the bare poses.

## Checking

```sh
xvfb-run -a -s "-screen 0 1920x1080x24" go run ./cmd/shot -w 1920 -h 1080 -o hd.png
xvfb-run -a -s "-screen 0 1280x720x24"  go run ./cmd/shot -o hd720.png
```

`-hdtest` sets down the trial stone, 1x to the left of the player and HD to
the right. `-subpixel=false` puts everything back on whole art pixels: shots
taken that way should match the old ones pixel for pixel (bar +-1 rounding),
except what you redrew. Look at both 1080p (crisp, 1:1) and 720p (scaled
down smoothly) for every sprite you change, and at a crop of it:
`-crop x,y,w,h` keeps a region, enlarged 4x.

## The ground

The ground is baked at `groundDens` (= `HDDensity`): a chunk of 8 x 8 tiles
is a 576 x 432 image, coloured image pixel by image pixel (`groundhd.go`
bakes, `groundart.go` holds the look of each kind of ground). It has two
looks, `GroundGreen` and `GroundWarm`, switched with `SetGroundStyle` (kept
in the settings file as `"ground": "warm"`; `shot -warm`). The shore image
the surf shader reads stays at density 1, as shaders run at art-pixel
resolution. `go test ./internal/client -bench BakeChunk` times a bake.

## The small plants and the ground cover

- **Drawings** (`coverart.go`): green grass, tall grass (the tufts that give
  fibre) and its stubble, leafy clumps, ferns, wild flowers, broad leaves
  and berry bushes, 4 to 6 drawings each, for each season (`coverPals`:
  summer's tall grass a muted straw, spring's green, autumn's golden,
  winter's bleached with snow on the tips; flowers open in spring, gone to
  seed in autumn). `coverFinish` gives each a soft foot: no outline along
  the bottom, the lowest pixels darker and fading into the ground round the
  root, and a soft contact shadow under it, so nothing stands on the grass
  like a sticker. A season's set is drawn the first time it is shown.
- **Each plant** is drawn mirrored or not, 13% bigger or smaller and a
  shade lighter, darker, warmer or cooler by where it grows, greener or
  yellower with the meadow's own patches (`coverLookAt`, `drawCover`).
- **Where they grow** (`undergrowth.go`, `plantChances`): noise fields, not
  an even chance: grass in drifts, flowers in patches of one kind, ferns and
  broad leaves under the trees (the shade of the trees in view, `noteShade`),
  glades where little grows. The tall grass, bushes, twigs and flowers that
  are things in the world are laid out the same way by the server in worlds
  of `sim.StyleGround` (`sim/groundcover.go`).
- **Life** (`coverlife.go`): the wind shears every plant about its foot in
  gusts running across the land, harder in a wind or a storm (`windSkew`);
  a plant bends away from whoever walks through it and springs back with a
  wobble (`bender`); tall grass hides the feet of someone standing in it
  (`drawFeetInGrass`); litter, needles or moss lie under each tree
  (`drawLitter`). Nothing is baked again for any of it.
- **The ground** (`coverground.go`, called from `grassGround` and
  `forestGround`): the sward thick and dark in places and thin in others,
  clover, moss, worn spots of earth showing through thinning grass, and in
  the woods moss cushions, dark duff and drifts of fallen leaves. All of it
  per pixel of the bake: the chunks cost what they did.
- `COVER_DUMP=dir go test ./internal/client -run DumpCover` writes every
  drawing three times as big on the ground it grows on (`cover_variants.png`)
  and the seasons (`cover_seasons.png`). `shot -walk dx,dy -gif N` has the
  player walk through the grass.

## The environment (trees, plants, rocks, buildings)

`mkUp(c, ax, ay)` makes the sprite of a thing that stands in the world
(`hdup.go`), by one of three roads, picked from the canvas:

- a canvas marked HD (`hdCanvas(w, h)`, or `markHD(newCanvas(...))` in a
  generator that works in pixels at a scale `s`, run at `s * hdK()`): the
  pixels are image pixels and so is the anchor. The brushes are in
  `hdenv.go` (leaves, bark, fronds), `hdplants.go`, `hdbuild.go`,
  `hdtrees.go`;
- `newEnvCanvas(w, h)`: a scaled canvas, for generators made only of
  `ellipse`, `line`, `polygon` and `rect` calls in art pixels;
- a plain density-1 canvas: refined by `upscale3` (outline thinned to one
  image pixel, staircases cut to diagonals, dither redone three times as
  fine). This is the fallback for what is not redrawn by hand yet.

`cmd/shot -envhd=false` (`SetEnvHD`) gives the original density-1 art back,
to compare.

## Sizes: the player is the yardstick

The standing player is `playerArtH` = 31 art pixels (93 px at 1080p), about
1.8 m. Trees stand 3-7 of them (`treeScale`, `trunkHalf`, `giantScale`),
bushes to the waist, doorways 1.2 (`wallArtH`, `hutDoorH`, `doorH` in
`desertart.go`, `caveMouthRise`). Draw a bigger thing bigger, with more
leaves, planks or stones, never by scaling the sprite; keep its anchor on
the ground point so tiles and collision stay. Check it on the scale sheet:

```sh
go run ./cmd/shot -w 1920 -h 1080 -scalesheet -o sheet.png   # heights printed too
go run ./cmd/shot -w 1920 -h 1080 -scalesheet -scaleonly build -o b.png
```

- What stands in front of the player or a beast fades while they are
  behind it (`occlude.go`): any sprite of 30 art pixels or more gets a coarse
  mask of its opaque pixels when its image is made (`noteCover`), and its
  mirror the mirrored mask. A beast that tall fades the same way while a
  player stands behind it (`seeThroughBeast`), but not for another beast,
  so a herd of long-necks does not fade itself.
- The beasts, in players: raptor 1.1, three-horn 1.7, plate-back 2.0,
  mammoth 2.6, rex 2.9, long-neck 6.5 (`-scaleonly creatures`). A painted
  dinosaur is made bigger by painting it finer over the same guide
  (`Species.K` in `internal/dinopaint`, `dinoGrow` in `dinorig.go`, which
  must agree): more scales and creases, not bigger pixels. Its rig, measured
  on the guide, is scaled to the painting when it is drawn (`rigAsDrawn`),
  and the seat, the shadow and the pet's heart through `beastScale`. The
  mammoth is drawn in code a little bigger (`mammothGrow`). The sim's
  hitboxes are unchanged. `shot -beasts "brachio,0,1.5;rex,4,2,left"` puts
  beasts round the player to see them in the world.
- The small animals, in players (`-scaleonly small`): rabbit sitting up
  0.32, crab 0.3, coiled snake 0.35, sitting monkey 0.44, toad about 0.17,
  fish 0.2; the cave spider 0.35 and the desert scorpion 1.0 are giants on
  purpose; the tribal warrior stands as tall as the player. They are drawn
  a little bigger than life, so they read at a glance, by `rabbitSize`,
  `crabSize`, `snakeSize`, `monkeySize` (`critsmall.go`) and `toadSize`
  (`toadart.go`): `newCritZoom` draws the same design that much smaller,
  and the shadow (`creatureShadowBase`) and the hop's bob follow.
- The big tree sets are drawn when first wanted (`lazyart.go`): the common
  ones on a goroutine just after start-up, the snow's, the swamp's and the
  woods' once the snapshot holds one. `shot -cost` prints what the art
  costs (bytes, start-up, trees drawn ahead).
- `shot -jungle`, `-at giant`, `-at hut` and `-move dx,dy` frame things for
  a look at their size.
- Trunks at HD go through `trunkHD` (`trunkhd.go`): a taper, a bow, a flared
  round foot, roots over the ground, a bark per kind (`barkKind`: furrowed,
  smooth, plates, scales, fibre, rings, shaggy, dead, and the cycad's
  diamond scars of fallen leaf bases, `cycadScar`) lit from the top left
  with a rim on the far edge, dead stubs, moss on the shaded side, and
  `dressFoot` laying grass, snow, sand or mud over the roots. Every bark
  ramp is in `barkOf`, which is how `isTrunk` tells wood from leaves.
  `shot -trunksheet` shows the foot of every kind of tree three times as big.
- The mangrove (`mangrove.go`) is wood laid out in the tree's own space (x
  across, z towards the viewer, h up) and seen from the game's angle: stilt
  roots leaving the trunk at different heights all round it, some forking or
  splitting into fingers, aerial roots from the crown and pencil roots in
  the mud, each a tube lit as a cylinder, with bark, rings, lenticels and
  the tide's algae and barnacles low down. Far ones are drawn first and
  darker, each with its shadow on the ground. Four variants (`mgSeeds`, the
  layouts picked by eye), each drawn in the water and on the sand
  (`mangroveSets`); the ripples and reflections under the roots that stand
  in the sea are drawn with the tree (`drawMangroveWater`). `MANGROVE_DUMP=dir
  go test ./internal/client -run DumpMangroves` writes sheets of every look
  and of the roots close up.
- Rocks, ore veins, coal seams, pebbles and the furnace (`rockart.go`) are
  built of convex blocks (each the inside of a few planes) seen from the
  game's angle: each pixel takes the face it meets first, so faces come out
  as flat planes and ore pieces sink into the stone. Then the steps of a
  pixel artist's rock: tones from COMFY52 (`comfy52`) by the light on each
  face, dark where blocks meet, an outline lighter on the lit side, cracks,
  lit lips and glints on the edges (`paint`). Boulders and pebbles take the
  stone of the ground they lie on (`rockGroundOf`: granite, sandstone,
  basalt, snow, cave); `fit` sets every rock to its height beside the
  player. `ROCK_DUMP=dir go test ./internal/client -run DumpRocks` writes a
  sheet of all of them and the steps one vein is drawn in.

- The leaf tent (`tentart.go`) is built in its own space and seen from the
  game's angle: an A-frame two tiles wide, its ridge running back (straight
  up the screen), each slope a plane (`slopeXY`, `slopeAB`) on which leaves
  are laid row by row from the foot up, tips down the slope, each a shape of
  that plane that casts a short shadow on what it lies over. Pixels keep a
  material and a light until the end (`tentPaint`), so folds and shadows add
  up before the tones are picked; leaves, bark and snow take clean bands,
  the bed of grass and the dark inside are dithered. Four thatches by the
  ground (`tentLeafOf`): fern, broad leaves, palm fronds, fir boughs under
  snow. It is two drawings, the inside and the shell: a sleeper is drawn
  between them (`drawSleeper`, `tentSleepDX`), so the poles and the thatch
  of the front come over their head and feet. `TENT_DUMP=dir go test
  ./internal/client -run DumpTent` writes every thatch three times as big,
  and with someone asleep in it; `shot -tentsheet dir` the sheet with the
  bag icon and a GIF of going to bed in it; `shot -tent -tentsleep` a camp.

## The sea and the cliffs

- **The sea's surface** (`seasurface.go`): the bake leaves the sea
  see-through, the more the deeper (`seaStops`), over a flat floor of deep
  blue; after the chunks, a shader runs over each chunk with sea in it at
  the canvas's own resolution (`drawShaderFine`), reading the chunk's alpha
  for the depth: soft swell bands, wave crests that rise and sink, glints
  (dimmer at night and under cloud: `seaLook`), and in wind and storms
  whitecaps and foam streaks down the snapshot's wind. Nothing is baked
  again. A swimmer's foam rings are dots a third of an art pixel
  (`fineDot`).
- **Reefs** (`terrainreef.go`): before a chunk's pixels are coloured, the
  coral round it is laid out in a layer of its own (`reefField`): clusters
  on a grid of about two tiles, one palette each (`coralPals`), heads of
  brain, staghorn, table, fan and soft coral and reef rock, drawn back to
  front with their shadows, then blurred a little; `reefPixel` lays the
  floor (sand patches, sea grass blades, rubble) and the coral tinted by
  the water. The fish are in the water shader (`terrainanim.go`, bit 4 of a
  sea texel: coral near).
- **Cliffs** (`terraincliff.go`): a front face reaches `wallFoot` of a tile
  down over the low ground and `wallLift` up over the high ground
  (`raise`), so a level is about 1.7 tiles of wall. The part over the low
  ground lower than `wallBand` pixels is baked again into a `front` band per
  row of a chunk's tiles and drawn in the sorted sprite pass
  (`wallDrawables`, sorted just before the middle of its row): figures at
  the very foot are behind the bottom stones, anything lower in front.
  Stones are rounded boxes in staggered courses (`blockAt`) shaded on the
  rock's ramp three times as fine (`fineRock`); plants grow from the cracks
  (`crackPlant`). Ramps (`rampPixel`, `terrainfeatures.go`) are flights of
  steps of the cliff's own rock.
- Look: `shot -exact -open -size 1024 -w 1920 -h 1080 -at
  cliffs-desert|cliffs-snow|cliffs-grass|ramp|reef`, `-swim -swimat deep
  -weather storm`, `-time 0.85`; `-bench 200 -frames 240` prints the frame
  time and the ground cache's memory.

## The natural caves

Worlds made since `sim.StyleCaves` have a cave system under each island
(`sim/cavesys.go`). Their chunks are baked by `cavebake.go` instead of the
ground bake, from the tiles and the cave cells the server sends (each tile's
chamber, rock and whether the main route runs over it):

- The rock's edge is the tiles' eased and pushed about by noise. Rock with
  floor below it shows a face `caveFaceH` (34 art pixels) tall inside its own
  tiles, built of facets lit from the upper left (`facePixel`): each rock
  (`caveMats`, COMFY52 tones) has its own, limestone with flowstone, mossy
  stone, sandstone in courses, basalt in columns, ice, mud. Above the faces
  the rock is the void of the roof with a thin rim (`topPixel`). Rock far
  from any cave is a cheap flat dark chunk (`deepRockChunk`).
- The floor (`floorPixel`): slabs, grit, pebbles with their shadows, damp
  patches and small puddles, shade by the walls and at the foot of faces,
  the worn main route.
- `cavezones.go` adds the themes: water, lava, ice and fords with their
  shores (`liquidPixel`, `shore`), each chamber's floor (`zoneFloor`,
  `tombFloor`) and the walls' accents (`faceAccent`: roots, icicles,
  crystals, veins of fire).
- What stands in the caves is drawn the first time it is wanted:
  `cavethings.go` (the hole down, the rope up, stalagmites by rock) and
  `caveprops.go` (crystals, giant mushrooms, webs, egg sacs, bones, the
  fossil, the paintings, the tomb's coffin, jars and pillars, roots,
  icicles, obsidian, the roost and its bats). The flat ones are drawn under
  everything (`drawCaveGround`); crystals and mushrooms light the dark.
- The light (`cavedark.go`): away from any light the caves are near black
  (`caveAmbient`, about 7% of full light, a little blue), at every hour.
  Each frame a light map at art-pixel resolution starts from that and
  adds every lamp with a soft edge (`caveLamps`: the held torch, torch
  stands, fires, crystals and glowing mushrooms, glowing plants, the
  daylight down the ways up, and lava from a tile mask); the scene is
  multiplied by it, then a little haze is added round the lamps. The
  eyes of spiders, bears and bats shine where it is dark
  (`drawCaveEyes`). The "Độ sáng" setting (Settings.Brightness, brightnessLevels) raises the dark a little, never enough to see without a torch.
- The map bakes each level as its own layer (`cavemap.go`).

`go test ./internal/client -bench BakeCaveChunk` times a cave chunk. Look at
a chamber with `go run ./cmd/shot -open -size 1024 -at cavezone:crystal`
(or lake, mushroom, nest, den, roost, fossil, paintings, tomb, lava,
frozen, flooded, stream), `-at caverock:limestone`, `-at caveprop:fossil`,
`-at cavein|cavedown|caveup`; the map with `-overlay map -reveal -mapzoom 3`.
For the dark, add `-nofire` (no campfire by the player), `-held 13` (a
torch in hand), `-cavelight 1|2` (the brightness setting), and `-frames 240`:
the cave fades in over the frames drawn, and a short run can be caught
before it has.

## The bosses and sea monsters

Titanoboa, the Megalodon, the Mosasaurus and the giant squid are drawn at
`HDDensity` as runs of frames (`bhAnim`, `bosshd.go`), each frame drawn the
first time it is shown and all of a beast's frames drawn ahead on other
threads once it is first seen (`hdWarm`). Bodies are tubes along a curve
(`hdTube`) or chains of knobs in a scaled frame (`hdFrameAt` + `bossPaint`);
the water they break is `hdOut` (foam, rings, spray, what is under seen
dimly). `bossHDSprite` (`bosses.go`) and `seaSprite` (`searender.go`) pick the
frame from the state and how long the beast has been in it (`stateAge`).
`BOSS_ANIM=dir go test ./internal/client -run DumpHDAnim -v` writes every
frame and what they cost; `shot -bosses -demobosses -bosspose N -bossage F`
shows them in the game.

## The temple and the atoll

The lairs of `sim.StyleLairs` (`sim/temple.go`, `sim/atoll.go`):

- **The temple's floor** (`templeart.go`): the courtyard's paving, the
  coping round the black pool and the pool's water are one per-pixel
  function (`templeLook.pavePixel`): courses of flags of their own depth
  and width, staggered, each with worn edges lit from the top left, a
  mottled face, cracks, lichen, moss creeping from the joints, flags gone
  towards the ragged rim; a ring of radial flags round the coping; the
  water black, the sky's grey in its far half. It is baked in blocks of 4 x
  4 tiles at HD the first time they are in view (two a frame) and let go
  when long out of view, and drawn in `drawLairFloor` (`lairfight.go`)
  under everything that stands.
- **The temple's stones** are rock scenes (`rockart.go`): `rkDrum`s
  stacked on a plinth and broken off at a slant for the colonnade's
  pillars, `rkLog`s lying in a row with the capital for a fallen column
  (one sprite across its three tiles), blocks for the serpent's head,
  fluted (`tplFlutes`) and mossed. The dead tree is the swamp's snag grown
  to seven men: `trunkHD` with `barkDead`, limbs forking three deep, beards
  of grey moss, its roots over the stones of a wall.
- **The reef** a player stands on is baked with the ground: `reefFlat`
  for a tile of sand with `VarReef` (coral heads lit from the top left,
  tide pools, weed, coralline, awash at the edge). The wreck's deck is one
  HD decal drawn flat (`genWreckDeck`); its mast stands (`genTallMast`).
- **What the fights leave** (`lairfight.go`): venom puddles
  (`venomPuddle`), the flood (vertex-shaded quads, ripples, rings round
  who wades), broken reef (`brokenReef`, rising back as it settles), the
  splash's rings, the shark's blood in the lagoon; the birds over the
  landmarks; spears in the air and a drawn spear's line.
- **The bosses' moves** pick their frames from the move and its phase
  (`bossSkillSprite`): `titanSweep` and `titanSlam` (`bosssnake.go`, the
  snake laid along headings with `tbBent`), `megaBeach` (`bossshark.go`),
  and the frames that were there; Titanoboa under its pool is a shadow
  with rings and bubbles (`drawTitanUnder`).

- **The camera in a fight** (`arenaCamera`): the arenas are bigger than
  the screen (the courtyard a rounded oblong of 40 by 31 tiles,
  `sim.ArenaQ`; the atoll 45 across to its shelf), and while the fight is
  on the view is kept inside (a little past the edge, more at the bottom
  under the hotbar), easing in and out over half a second, after the
  boss's entrance and before the shake; it is drawn back to 0.9 (drawn
  for a screen a ninth bigger and shrunk, `lf.pull`). A red arrow at the
  edge points to the boss when it is off the screen (`drawBossArrow`).

Look: `go run ./cmd/shot -exact -w 1920 -h 1080 -size 1024 -bossfight
titan|shark|map -o dir/x.png` plays the scenes in a real game and writes
the stills, the GIFs and a still of every tenth GIF frame in `dir/review`.

## Item icons

The old item icons are 16 x 16 pixel maps (`itemicons.go` and the
`gen*Icons` of each file), scaled up whole times in a slot: an icon pixel
is a block of 6 screen pixels in a bag slot at 1080p. HD icons (`iconhd.go`) are painted
in code at the screen's own resolution instead, one icon pixel to one
screen pixel, like the world.

- **Design box.** A painter draws in a 16 x 16 box of design units, the box
  the old icon filled, so the HD icon takes its place and size exactly. It
  is baked once per size in screen pixels, the first time it is shown (and
  ahead, on another goroutine, at the sizes of the screen's UI scale): 96
  in a bag or hotbar slot at 1080p, 64 in a chest and on a recipe card, 32
  for a recipe's ingredients; half that at 720p, twice at 4K. Under 40
  pixels it is painted bigger and shrunk; at 16 or less the old icon is
  drawn (it was made for 16). The box sits inside a margin as wide as the
  outline, so nothing is cut off.
- **On the screen.** `g.drawItem(dst, item, x, y, size, alpha)` (UI
  pixels, on the interface's `uiDst`) draws an item's icon: HD if it has
  one, straight on the screen at its true size, else the old one, scaled
  as before. What covers it (a count, a badge) is simply drawn after it.
  In the world (a pile on the ground, food or a bowl in the
  hand) an item shows its HD icon at the world's density, or, without
  one, its old icon redrawn three times as fine (`iconFine`); tools and
  weapons in the hand have painters of their own (see "Held items").
- **Off.** `shot -iconhd=false` (`SetIconHD`) gives every item its old icon.

### The look (the pilot set, `iconhd_pilot.go`)

- A clear silhouette that reads at slot size: fill the box (about 1 to 15),
  few big masses, small detail only where it says what the thing is.
- Light from the top left (`lightDir`), the world's. Each material is a
  ramp of 4 or 5 tones picked in clean bands (`pick`): shadow, mid, light,
  a highlight; glints are single `icGlint` marks, never gradients.
- Outline: the bake puts it on (`outlineIcon`), 1 pixel up to about 100
  across, 2 at 4K. Near ink where the edge faces away from the light, the
  colour it borders darkened where it faces the light. A part painted over
  another darkens a line on the one beneath (`contact`), so a lashing, an
  end of a log, a berry stand apart.
- A soft shadow falls down and to the right (`dropShadow`); glow and steam
  (`soft`) go over everything, without outline.
- Materials, each with its tell: wood grain streaks along a rod and rings
  on a cut end (`shRod`, `shRing`); stone in flat faces with lit lips and
  dark creases (`shFacet`), knapped stone and flint in shallow scallops
  (`knap`); flint glassy with a glint, in a chalky rind; meat red with fat
  flecks and a fat rind, roast meat with grill bands; berries round with a
  sharp glint and bounced light below (`shSphere`); cord and rope twisted
  (`shCord`, `binding`); hide flat and mottled with fur at the edge.
- Tools lie from the bottom left to the top right, the working end up, its
  edge towards the light; food is turned a little.
- Colours from the old icons' ramps (`iiWood`, `iiLeaf`, `ciPot`...) and,
  for stone, the COMFY52 tones of the boulders (`rkGraniteMat`): an icon
  looks like the thing on the ground.

### Adding an icon

1. Write a painter, `func icThing(p *icv)`, in a file of its own group
   (say `iconhd_ores.go`), and register it in that file's `init`:
   `regIcon(sim.ItemThing, icThing)`. Nothing else changes: every UI that
   shows the item picks it up, and the old icon stays as the fallback.
2. Paint back to front, in design units: `p.fill(shape, shader)` a new
   part, `p.add` more of the last part (no line between), `p.on` marks on
   what is there, `p.under` a part behind, `p.soft` glow. Shapes are those
   of `hdfig.go` (`sCap`, `sEll`, `sPoly`, `sPath`, `.or`, `.and`, `.sub`,
   `.grow`, `.moved`) and `iEll`, `iBlend`, `sTaper`, `sRing`, `.rough`;
   `frameOf` lays out a tool's head in its haft's frame. A shader gets the
   pixel's design point (`f.x`, `f.y`), its depth in the part (`f.d`) and
   the edge's outward normal (`f.gx`, `f.gy`), and returns a colour.
3. Look at it: `ICON_DUMP=dir go test ./internal/client -run DumpIcons -v`
   writes `icons_hd.png` (each icon at 96 and three times bigger, in a
   second or so; `ICON_ONLY=<item number>` for one). In the game:
   `go run ./cmd/shot -exact -w 1920 -h 1080 -iconsheet -o s.png` (old and
   new in 1080p hotbar slots, the smaller sizes, and x3; 30 icons to a
   page, `-iconpage 1` and on for the rest), and
   `-overlay recipes -bagfrom 1` for the bag with its recipe cards.
   `-exact` matters on a desktop shorter than 1080: without it the window
   is cut short and the interface drawn at scale 1.
4. `go test ./internal/client -run HDIcons` bakes every icon at every size
   and fails if one covers too little of its box or touches its edge.

A pilot icon took 10 to 20 minutes, most of it looking at it at 96 and x3
and moving a few numbers; a bake costs 1 to 8 ms at 96 pixels.

## The interface

The interface is laid out in UI pixels, as it always was, and drawn at
the screen's own resolution (`uiscale.go`): `beginUI` hands out a
`uiDst`, the screen seen in UI pixels, `k` screen pixels to a UI pixel
(automatic: the screen's height over 540 in quarters, so 1.25 at 720p, 2
at 1080p, 2.5 at 1440p, 4 at 4K; or the "Cỡ giao diện" setting, 100% to
400%). `k` need not be whole: panels and bars are baked at the size they
land at, put on whole screen pixels, and text is set at its true size, so
250% is as sharp as 200%. Layout,
slot sizes and the mouse (`uiCursor`) stay in UI pixels; only the drawing
is finer. A `uiDst` is a `drawTarget`, so `fillRect`, `strokeLine`,
`fillCircle`, `fillPath`, `drawSprite`, `drawAt` draw on it in UI pixels;
`d.hair()` is one screen pixel, `d.clip` cuts it to a rectangle, `d.view`
lays out a panel at its own size and shrinks it (the difficulty picker).

- **Type** (`uifont.go`). Nunito (SIL OFL 1.1, `fonts/`, see `CREDITS.md`),
  set from its variable font: weight 720 for text, heavier for counts and
  badges, 930 for titles; Go Bold behind it for the arrows Nunito lacks.
  `drawText(dst, s, x, y, colour, align, scale)` keeps the bitmap font's
  layout (a line is 16 UI pixels at scale 1, its top at `y`); `uiText`
  and `uiTextW` take an effect: `fxShadow` (a soft edge and drop shadow,
  the default), `fxOutline` (a fuller halo, over the world), `fxPlain`
  (ink on parchment: `ink`, `inkOf`). Each string is baked once with its
  effect, white, and tinted when drawn; `textWidth` measures at the size
  drawn now. `fitText` shortens a string with "…".
- **Panels** (`uipanel.go`). One shader paints every style into an image as
  big as it is on the screen, with its shadow: `psHide` (dark hide, a
  stitched leather rim: `panel`), `psWood` (a wooden frame with bone studs:
  `drawFrame`), `psParchment` (`parchment`), `psStone` (the boss),
  `psWell` (slots, bar tracks: `well`), `psPlate` (`plate`, `plateA`),
  `psLeather` (`tab`: buttons, cards, chips), `psGlow` (`glow`). A look's
  colours are straight (the alpha is how much shows). A bake is kept while
  drawn and let go a few seconds after; what fades or pulses is baked once
  and drawn fainter (`drawPanelAlpha`). Pieces made of shapes that do not
  change (a key badge, the minimap's rim, the day's bar) go through
  `drawBaked`, which keeps them the same way.
- **Bars** (`uitheme.go`): `well` for the track, `drawBarFill` for the
  fill (a shader: lit like something round, a slow sheen, a wavering end),
  with `barFX`: the share it had a moment ago draining away pale, and a
  flash; `barTrail` works them out. A vital that runs low glows.
- **Pictures** (`uiicons.go`): the vitals' heart and the status chips'
  icons are painters like the item icons', baked per size.
- **Maps** (`uimap.go`): ground not yet walked is faded paper with
  hatching (`fogSrc`) on both maps; the minimap is drawn at the screen's
  resolution and cut round (`maskSrc`), a circle in the ring of the day.
- **The HUD** (`hudlayout.go`, `hud.go`, `hudtoasts.go`) is kept small so
  the world is what is seen, 12 UI pixels from the edges: thin vital bars
  (11 px, the value small inside) on a see-through hide (`hudPanel`),
  round status chips under them (a buff's rim runs down like a clock), the
  long line of a status said once under the chips as it begins; top right
  the round minimap in the ring of the day (a sun or moon rides it), one
  line of day, season and weather under it; top middle a slim boss bar,
  the notices under it (three at most, fading by themselves); over the
  hotbar one line, the held item's name for a moment, else the one hint.
  Every piece's place comes from `hudlayout.go`, and
  `TestHUDLayoutKeepsClear` checks they keep clear of each other at every
  screen size and UI size. A chip or the day's line names itself under the
  pointer.
- **Over the world** (`uilabels.go`): one label at a time, for what the
  pointer is on, on a soft plate that fades in and out over 140 ms; what is
  out of reach gets a small hint; the target is marked by thin soft
  corners (`drawCorners`). Other players' names are `nameTag`s.
- **Checking.** `go test ./internal/client -run TestInterfaceScreens` draws
  every screen at 720p, 1080p and 4K (in a child process with a small
  window: the art needs Ebitengine's loop) and checks through `uiProbe`
  that every string is inside its box and over no other. Draw a new box
  that holds text with one of the pieces above (they tell the probe), or
  call `noteBox`; a full-screen dim is `dimScreen`. The screenshot tool
  shows them: `-overlay bag|recipes|craft|map|palette`, `-chest`,
  `-cook -cookui pot`, `-bosses -demobosses`, `-menu`, `-create`,
  `-devconsole`, `-freebuild`, `-status`, `-skillstatus`, `-buffs`,
  `-notices "a|b|c"`, `-aim`.

## Held items

Tools and weapons in the hand are painted at `HDDensity` in the look of
the HD icons (`heldhd.go` draws them, `heldhd_art.go` holds the painters):
the axe, pickaxe and hoe of every tier, the knives (swords) of flint,
jade, agate and obsidian, the spear and the ivory spear, the war axe, the
torch and the fishing rod. Anything else in the hand, and every pile on
the ground, is its HD icon at the world's density, or its old icon redrawn
three times as fine (`iconFine`).

- **Its own frame.** A held item is painted in art pixels with the grip,
  where the fist closes, at (0, 0) and the working end straight up (-y),
  the edge forward (+x). `heldDef` gives the painter, the box it paints
  in, its `scale` (art pixels to the painter's design units), where its
  head is (`head`, `reach`: a swing's trail), how it is `carry`'d, and
  `grip`, the gap left in the haft at the grip:
  the fist, drawn before the item, shows through it, so the hand closes
  round the haft instead of the haft lying over the hand.
- **Its real size** beside the 31-pixel figure (`TestHeldSizes`): an axe,
  a pick or a hoe 0.4 to 0.45 of its height (forearm and hand), a knife
  about a quarter, the spear and the ivory spear 1.2, the torch a third
  (its flame `fireHand`, smaller than a torch's on its stand), the rod
  1.0, the war axe a half. A painter is set to it by its `scale`, not
  resampled, so the heads keep their detail; the knife and the spears are
  drawn at their size.
- **Turned before it is painted.** Swings turn tools through every angle,
  so each item is painted anew at each of `heldTurns` (64) angles: every
  shape is turned first (`hpv.xf`), so edges stay clean, the outline stays
  one pixel and the light stays in the top left at every angle. Painting
  is three times finer and shrunk (`heldSS`). Grain, knapping scars and
  agate's bands are taken in the item's frame (`hpv.loc`), so they stay
  on the item as it turns; normals of the item's frame are turned onto the
  canvas (`hpv.lit`, `hpv.rod`). Facing left the image is mirrored, as
  the figure is. An item's angles are painted the first time it is shown,
  ahead on another goroutine; an angle wanted before then is painted on
  the spot (about 5 ms). One angle is 1 to 27 KB; all 64 angles of every
  held item would be about 8 MB, a session pays for what is held.
- **Stone by tier** (`heldMats`): stone knapped in broad matt scars, flint
  in finer scars with a waxy gloss, jade and agate ground smooth and
  polished (agate banded), obsidian in big glassy scars. `face` shades a
  face of a head from the normal it leans to, bevels it to its edge, and
  puts a glint only where an edge or a ridge mirrors the sun to the eye.
- **Carried** (`drawCarried`), standing, idling, walking or running: the
  arm that holds something swings with the stride, less and straighter
  (`upWalkHold`, `upRunHold`). A tool hangs from the hand, its head down
  and forward, the edge forward (mirrored), turning with the forearm
  (`hangLean`, `hangFollow`); seen from the front or the back it hangs
  outwards behind the body, foreshortened (`heldLowQ`, painted when first
  wanted), its head beside the hip. A spear or the rod is carried along
  the body, the point forward and 30 degrees up (`poleUp`), upright at
  the side from the front or the back. A torch is held up and forwards,
  the flame over the shoulder: its arm stays raised (`upTorch` in the
  upper half's frame, `torchArm`) whatever the other does.
- **In the hand.** `drawHeldAt` puts the grip on the hand of the jointed
  figure's frame and tips the item by `heldCarry` plus what the forearm
  turns (`heldTurn`); the chop, the slash and the war axe's cleave turn
  it about the grip on their own clocks. The item goes behind the body
  when the hand is (facing up, or the far hand), and behind the head with
  a hand raised beside it, seen from the side, or a tool raised over the
  head, seen from behind (`figDraw.tool`). Reaching up for fruit, the
  other hand takes it (`figDraw.offHand`). Swimmers and sleepers hold
  nothing; riders keep it.
- **Trails.** The slash, the cleave and the chop leave an arc drawn at the
  canvas's own resolution (`swingArc`, triangles on the full canvas), as
  is a fishing line from the rod's tip (`fineStrip`). The torch's flame
  is painted apart in `torchFlameN` frames and stands upright on the
  turned torch; where the flame and the rod's tip were drawn is noted for
  the torch's light and the line (`markHeld`).
- **Off.** `shot -heldhd=false` (`SetHeldHD`) gives the old density-1
  sprites and icons back, to compare.

Look at them: `go run ./cmd/shot -exact -w 1920 -h 1080 -heldsheet dir`
writes `held_compare.png` (each item in the caveman's hand, old and new,
at 1080p and three times as big about the hand), `held_swings.gif` (the
chop, the slash, the thrust and a walk with a torch, every facing),
`held_motions_*.png` (the motions that keep something in the hand) and
prints what painting them costs. `shot -heldfit dir` writes `held_size.png` (each
tool in hand standing, every facing) and GIFs of a run from behind and
from the side (`held_run_back.gif`, `held_run_side.gif`). `HELD_DUMP=dir go test ./internal/client
-run DumpHeld` writes every item at a few angles four times as big
(`HELD_ONLY=6,32` for some), and `go test ./internal/client -run Held`
paints every angle of every item and checks each is whole.
