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
- UI (bag icons, the character screen) draws on the screen: `drawSprite`
  sizes HD sprites right there too. Item icons have their own pipeline:
  see "Item icons" below.
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

What it costs: every frame of every motion of one look is 1632
half-frames, 14 MB if all were shown, 1.3 ms each to draw; they are drawn
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
  mask of its opaque pixels when its image is made (`noteCover`).
- The big tree sets are drawn when first wanted (`lazyart.go`): the common
  ones on a goroutine just after start-up, the snow's, the swamp's and the
  woods' once the snapshot holds one. `shot -cost` prints what the art
  costs (bytes, start-up, trees drawn ahead).
- `shot -jungle`, `-at giant`, `-at hut` and `-move dx,dy` frame things for
  a look at their size.
- Trunks at HD go through `trunkHD` (`trunkhd.go`): a taper, a bow, a flared
  round foot, roots over the ground, a bark per kind (`barkKind`: furrowed,
  smooth, plates, scales, fibre, rings, shaggy, dead) lit from the top left
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
  everything (`drawCaveGround`); crystals and mushrooms light the dark
  (`caveGlows`, `cavelights.go`).
- The map bakes each level as its own layer (`cavemap.go`).

`go test ./internal/client -bench BakeCaveChunk` times a cave chunk. Look at
a chamber with `go run ./cmd/shot -open -size 1024 -at cavezone:crystal`
(or lake, mushroom, nest, den, roost, fossil, paintings, tomb, lava,
frozen, flooded, stream), `-at caverock:limestone`, `-at caveprop:fossil`,
`-at cavein|cavedown|caveup`; the map with `-overlay map -reveal -mapzoom 3`.

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

## Item icons

The old item icons are 16 x 16 pixel maps (`itemicons.go` and the
`gen*Icons` of each file), drawn on the interface's canvas, which is blown
up 2 times at 1080p and 4 at 4K (`uiscale.go`): an icon pixel became a
block of 6 screen pixels in a bag slot. HD icons (`iconhd.go`) are painted
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
  pixels) draws an item's icon: HD if it has one, else the old one, scaled
  as before. On the UI canvas an HD icon is not drawn there: it waits, and
  `uiBreak` puts the canvas, then the icons, then `uiTop` on the screen.
  What must cover an icon (a stack's count, a key badge, the torch's bar)
  is drawn on `overIcons(dst)`. `uiBreak` is called by `dim`, the notice
  and the dev console, and `endUI` ends with it: call it before anything new
  that is drawn over item icons (a tooltip, a dialog), or it will end up
  under them. World uses (a drop on the ground, a held item) still take
  the old icons from `g.art.icons`.
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
   new in 1080p hotbar slots, the smaller sizes, and x3), and
   `-overlay recipes -bagfrom 1` for the bag with its recipe cards.
   `-exact` matters on a desktop shorter than 1080: without it the window
   is cut short and the interface drawn at scale 1.
4. `go test ./internal/client -run HDIcons` bakes every icon at every size
   and fails if one covers too little of its box or touches its edge.

A pilot icon took 10 to 20 minutes, most of it looking at it at 96 and x3
and moving a few numbers; a bake costs 1 to 8 ms at 96 pixels.
