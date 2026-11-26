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
6. Paint, add the name to `PAINTED` in `gen.py`, and look:

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
