# The HD dinosaurs

`dinos/hd/<name>.png` is the drawing `dinos/<name>.png` three times as fine
(`HDDensity`), with the same layout, so the rig (`dinorigs.go`, measured in
pixels of the small drawing) moves the same anatomy at either density.

Two ways make them:

- **Painted** (`rex`, `trike`): `go run ./cmd/dinopaint <name>` paints the
  drawing over again from a description in `internal/dinopaint/<name>.go`.
  The small drawing is the guide for the silhouette and the colour regions;
  everything else is drawn: masses lit as forms, muscles, scales, knobs,
  creases, eyes, teeth, claws, horns.
- **Filtered** (the rest, for now): `gen.py` upscales and textures the
  drawing. It skips the species in its `PAINTED` set.

**Bigger than the guide.** A description's `K` paints it finer than three
image pixels to a guide pixel: the three-horn at 3.33, the plate-back at
3.6, the rex at 3.75, the long-neck at 5.5. The painting is then `K / 3`
times as big in the game, with that much more detail (the guide's shapes,
muscles and parts are in guide pixels; scales, outlines and fine marks are
in image pixels, so they stay as fine). It is padded to whole art pixels
on the right and at the bottom. The game scales the rig with it
(`dinoGrow` in `dinorig.go`, which must say `K / 3`).

## Adding a species to the painter

Copy `rex.go` (a biped) or `trike.go` (a quadruped with a frill and horns)
in `internal/dinopaint` and change it. Everything is in pixels of the
guide (`dinos/<name>.png`): open it enlarged with a grid to measure.

1. **The rig first.** Read the species in `internal/client/dinorigs.go`:
   its legs' columns (`x0..x1`, from row `top`) and joints, the head's
   pivot and cut line, the jaw line. Claws, nails and toes must stay inside
   their leg's columns or they will not move with the foot; whatever turns
   with the head must be on the head's side of its cut.
2. **Materials.** One per colour region of the drawing: `Seeds` are guide
   colours (list them with a palette dump; the `-debug` maps show what each
   pixel became), `Ramp` 8 colours from deep shadow to highlight, shifted
   warm in the light and cool/purple in the dark. Thin markings keep their
   width with `Bias`; `Smooth` rounds them. `Region` paints a material on by
   shape (eyespots, a beak); `Recolor` fixes the drawing's own shading
   taken for a marking. `Ink` is the luminance under which a guide pixel is
   outline. `GuideDetail` (0.3-0.6) carries the drawing's small marks and
   inner lines into the light.
3. **Parts**, back to front: far legs (`Far`), body, head pieces, near legs.
   A pixel takes the frontmost part whose `Shape` holds it, so draw each
   part whole, also where it is hidden. `Round` is how far in from its edge
   it rounds over (`RoundAt` to vary it); `Muscles` are bulges (`Mu`). A
   near leg grows `Onto` the body: outside `Seam` its top runs smoothly
   into the hip, inside `Seam` its edge is a crease with a shadow.
4. **Skin.** Scale radius on the flank, back and belly (image pixels);
   `Osteoderms` rows along the top edge (`Crest` raises the outline over
   them); `Folds` at the throat, knees, ankles and toes.
5. **Details** (drawn last): `Eye`, `Teeth` + `Lip` (upper teeth end above
   the rig's jaw line, which then sets `drawnTeeth`), `Nostril`, `Claw`,
   `Horn`, `Nail`. Cut a guide's thin horns out of the silhouette (`Remove`)
   and paint them again.
6. **Under the head** (`RegisterUnder`, `under.go`), for a head that hides
   the body: a frill over the shoulders. The painter then also paints
   `<name>_under.png`, the body without the head's parts, whole where they
   hid it (add a part, a neck say, that only shows there). When the rig
   turns the head (or the neck), only the head's own pixels turn, the
   body's stay put on either side of the cut, and the painted body shows
   where the head was (`dinounder.go`) instead of the head swept round to
   fill the joint. Every pixel of the head must be in a head part's
   `Shape`. With no joint filled, a long neck bends about a pivot on the
   outside of the bend (the top of the neck, to lower it), so the bend
   folds over the throat instead of opening a gap.
   **Legs** (`RegisterLegs`, `legmask.go`): name each rig leg's parts, in the
   rig's order, and the painter also writes `<name>_legs.png` (which pixels
   are which leg) and `<name>_legsunder.png` (the far legs whole where the
   near ones hide them). The rig (`dinolegmask.go`) then cuts each leg along
   its painted outline instead of in columns, so a stride leaves no chips,
   straight edges or holes. **Thighs** (`RegisterThighs`, `thigh.go`), which
   every walking species has now, go further: each leg is cut whole from
   the hip or shoulder, its thigh or upper arm (`Thigh`, a shape per leg)
   bending from the body as the stride swings it. The painter writes the
   legs' mask (blue: what of the body is in front of the near legs,
   `Front`/`FrontParts`, and what turns with the thigh), the legs whole
   where something hides them (`_legsunder.png`, the near thigh's top under
   a folded wing, the far legs up into the hip), and the body where the legs
   hide it (`_legsbody.png`, with the species' `Coat`), which shows as a
   thigh swings off it (`dinolegthigh.go`). More helpers for a description's details:
   `Plate` (a stegosaur's plates), `RimSpikes` (a frill's), `Gloss` (wet
   skin), `Fuzz` (a fuzzy coat), `Trace`/`Clear` (drawn water), `Tint`.
   Armour (`armour.go`, the ankylosaur's): `Scutes` lays bony plates edge
   to edge in bands down from the back, keeled, rimmed with pale bone, the
   skin dark between them; it changes the light and the materials, so it
   comes first among the details, then `Redraw`. Spikes on the body painted
   between `Mark` and `Claim` stay with the body when a leg below steps.
7. Paint, add the name to `PAINTED` in `gen.py`, and look:

   ```sh
   go run ./cmd/dinopaint -debug /tmp/dbg <name>     # also the material, part and light maps
   DINO_SPECIES=<name> DINO_REVIEW=/tmp/rev go test ./internal/client -run DumpDinoReview
   go run ./cmd/shot -w 1920 -h 1080 -aim 0,3 -dinos -coast -o shot.png
   ```

   The review writes every pose of the rig and GIFs of the walk and the
   blow: check the joints (hips, knees, the neck) and the open mouth.
   `go test ./internal/dinopaint` fails if a PNG is older than its
   description.

A species takes about an hour or two: measuring the parts and muscles is
most of it, then a few rounds of looking at the painting and the poses.

## Feathers and wings

The feathered hunters (`dakota`, `achero`, `troodon`), `ornitho`, `anzu`
and the pterosaurs (`ptero_fly`, `ptero_perch`, `quetzal`, `quetzal_fly`)
are painted too, with helpers called first thing in their `Details`:

- **`Plumage`** (`feather.go`): tracts of feathers, back to front, each
  laid along a flow (`coords.go`: `Along` a line, `Away` from one, a `Fan`
  from a point, a `Dir`). The feathers lie in rows, each over the root of
  the ones behind, so what shows is a scallop: lit over its shaft
  (`Shaft`, `Camber`), shading the feathers under it, the outline made of
  their tips (`Fluff`; also at the picture's edge, which a tail may touch).
  `Snap` lays the drawing's mottling feather by feather; `TipMat` with
  `TipGuide` turns the drawing's pale marks into pale feather tips
  (`TipElse`: what the marks become elsewhere). `Hair` makes fine strands
  instead: down, pycnofibres. What no feather covers keeps its scales.
- **`Membrane`** (`membrane.go`): a wing's skin, nearly flat, bellied
  behind the leading edge (`Camber`), its fibres fanning out (`Fibres`),
  darker towards the tip (`Distal`), light through its thin free edge
  (`Glow`, `GlowRamp`, `Free`), and `Bones` read through it as lit tubes.
- **`Plane`** (`plane.go`): a crest or casque lit as a flat plate with a
  bevelled rim and growth lines.
- **`Beak`** (`beak.go`): keratin over a beak: a gloss under the ridge,
  grain, the mouth's line, the lower beak (`Lower`), a nostril.

The night hunter's eye is painted in the drawing's two ambers only, which
the game finds to draw it again in every pose (`troodon.go`).
