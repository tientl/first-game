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
  sizes HD sprites right there too. Item icons are plain `*ebiten.Image`s,
  not sprites: they need their own density handling when redrawn.
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
  of `dinos/` redrawn 3x as fine by `dinos/hd/gen.py` (rerun it after
  changing a drawing or its settings there). The rig (`dinorig.go`) is still
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
- Not density-aware yet: the sprite export in `export.go`, the fallen body
  (`bodyart.go`) and the map's head marker, which still use the density-1
  figure.

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
