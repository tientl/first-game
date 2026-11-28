# Playtest check (PLAN.md: "Kiểm tra trước khi cho chơi thử")

Measured on branch `claude/playtest` (open-world f11dc97).

**Machine and caveat.** Windows 11, i5-11400H (6 cores / 12 threads), 16 GB RAM, RTX 3050 Laptop + Intel UHD, 1920x1080 frame, vsync off, every frame read back from the GPU. **Five agents shared this machine the whole time** (available RAM dipped to 0.5 GB, two of my runs died with `out of memory` / `VirtualAlloc errno 1455` / a DirectX `CreateTexture2D` failure). Frame times below are therefore noisy and pessimistic; runs made when the machine was quiet are marked. Treat memory *trends* as solid and absolute milliseconds as rough.

## Tools written

- `cmd/soak` (new): plays the real `client.Game` against an in-process dev server through a `LocalConn`, along a script. `-mode scenes` = frame time per heavy scene; `-mode soak` = a long tour (walks, teleport hops across the map, caves, sea, villages, boss lairs, crowds of beasts, night/storm). Prints markdown tables (median/p95/p99/max per scene, the first 20 frames, every frame over 50 ms, Go heap / live heap after GC / working set / private bytes, art bytes, ground chunks, cache sizes). `-memprofile f` writes a heap profile. Usage in the file header. Windows memory counters in `procmem_windows.go`.
- `client.CacheStats()` (`internal/client/cachestats.go`): entries and bytes of every process-wide client cache.
- `TestLoadTimes` (`internal/server/loadtimes_test.go`, `LOADTIMES=2048 go test -run TestLoadTimes -v ./internal/server`).
- `TestEverySaveVersionLoadsTicksAndSavesAgain` (`internal/sim/saveversions_test.go`).
- Existing: `cmd/shot -bench N -cost` (used for the first sanity run).

## 1. Memory

Single scene (`shot -open -size 1024 -bench 300 -cost`, standing at spawn): 83 MB of art images drawn in code, start-up set 3.2 s, trees drawn ahead 2.4 s, 30 ground chunks = 28.5 MB, Go heap 484 MB in use (1.4 GB from the OS right after world generation).

Scene tour at 1024 (`soak -mode scenes -gc`, quiet-ish run, live heap = after forced GC):

| after scene | live Go heap MB | art MB (cumulative) | ground chunks | rig frames | critter frames |
|---|---|---|---|---|---|
| baseline (world made, no frame yet) | 60 | 0 | 0 | 0 | 0 |
| spawn, day | 765 | 325 | 30 | 34 | 195 |
| dense forest | 1064 | 334 | 60 | 34 | 251 |
| open sea | 1252 | 338 | 72 | 34 | 259 |
| village, night | 1266 | 341 | 72 | 62 | 284 |
| forest, storm | 1980 | 340 | 72 | 62 | 292 |
| many dinosaurs | 2383 | 357 | 72 | 218 | 292 |
| cave | 3397 | 362 | 84 | 240 | 298 |

(An earlier run, with the machine busier, showed 600 MB to 2.0 GB live over 3 minutes, working set 2.3 GB, private bytes 4.4 GB, and a 6 GB heap / 9.8 GB private without forcing GC.)

Cache audit (`CacheStats` at the end of the tour): ground chunks 72-84 (68-86 MB, evicted by `evict`, **bounded**); dino rig frames 186-240 entries (27-41 MB, one per species x pose: bounded by the species list but the biggest sprite cache; grows when many species are met); critter frames 288-298 (19 MB); mirror images 19-25 (7-19 MB); panel/text/icon bakes under 6 MB, evicted; cover masks 555 (tiny); all ID-keyed clocks (boss, blow, strike, skill) are capped at 256-512 and cleared. Ground chunks, trees, creature sprites, voices and fire frames do **not** grow with distance travelled.

Findings:

1. **Live Go heap grows ~1 GB per few minutes and the cause is not a client cache.** Heap profiles (`-memprofile`) at 1 minute and 100 s show the live heap is dominated by Ebitengine internals: `graphics.(*bytesPool).get` from `atlas.Image.writePixels` (2.3 GB at 100 s) and `graphicscommand.uint32sBuffer.alloc` (0.9 GB), i.e. pixel uploads and draw commands held by the engine's command queue and byte pool. The game's own retained data was under 100 MB apart from item 2. I could not tell whether this is a real unbounded pool in Ebitengine 2.10.6 (its `bytesPool` keeps every freed buffer and picks first-fit) or an artefact of my harness (vsync off, tens of ms frames, a 1-pixel `ReadPixels` per frame, teleports that bake dozens of chunks at once). **A 10-minute soak could not be completed** (the machine ran out of commit memory three times). This is the main thing to re-check on a quiet machine: play the real `cmd/game` for 10 minutes and watch Task Manager. If it is real, the fix is in how chunks are baked/uploaded (fewer, smaller `WritePixels`), or a periodic `debug.FreeOSMemory`.
2. **Boss and sea-monster art is held as CPU pixel canvases.** `hdWarm` (bosshd.go) draws every frame of every move of a boss ahead on worker goroutines, and each canvas stays in memory as pixels until that frame is first shown: 233 MB live after one minute (Titanoboa poses 117 MB, mammoth king 49 MB, Megalodon/Mosasaurus/Kraken the rest) in a session that merely passed sea monsters. I tried turning the warmed canvases into images a few per frame; that only moves the 230 MB to GPU memory (art bytes jumped 88 to 325 MB), so I **reverted** it. Recommendation: do not warm every move; warm the idle/chase moves only, and draw the rest on first sight.
3. **Leak fixed:** `voiceMemory.engaged` (tribe warriors' last war cry, `tribesound.go`) was never pruned; it now drops entries older than a minute past 256. Small, but the only unbounded per-creature-ID map found.
4. Art drawn in code totals 83 MB at the first frame, but 325 MB once the first scene has run (trees drawn ahead, HD sets); creature/boss art adds 40-50 MB each time a new family is met.

## 2. Frame time (1920x1080, 1024 world, vsync off, GPU read back)

`soak -mode scenes`, quality 100, zoom 0, one scene right after another, machine shared. Median / p99 / max in ms; "hitch" is the longest of the first 20 frames after teleporting in (art drawn the first time it is shown, ground chunks baked).

| scene | median | p99 | max | longest first-20 |
|---|---|---|---|---|
| spawn, day | 24.7 | 99 | 150 | 84 |
| dense forest | 14.4 | 131 | 1453 | 1453 |
| beach and reef | 16.8 | 106 | 693 | 693 |
| open sea (boat) | 22.9 | 134 | 1912 | 36 (hitch later, #26) |
| village, day | 54.0 | 292 | 914 | 30 |
| village, night (fires) | 51.5 | 214 | 216 | 214 |
| forest, rain | 50.6 | 161 | 207 | 102 |
| forest, storm, dusk | 50.9 | 109 | 131 | 85 |
| many dinosaurs (about 50) | 53.2 | 171 | 255 | 255 |
| cave | 22.6 | 79 | 655 | 655 |

Quality 75, zoom 0 (same tour, 6 s per scene, noisier): spawn 16.0 / 54 / 169, dense forest 10.2 / 207 / 1441, beach and reef 9.4 / 44 / 429, open sea 15.7 / 123 / 1495, village day 53.5, village night 52.4 / 124, forest rain 51.5 / 165, storm 51.5 / 115, dinosaurs 61.7 / 349, cave 21.5 / 84 / 493, cave with crystals 22.5 / 68 / 334, lava 27.2 / 101 / 533. Quality 75 is not clearly cheaper in these runs: the scenes at 50 ms are in the second half of the tour, when the machine was most loaded and the heap was 1-3 GB. **Zoom +1 was not measured** (machine out of memory); first run with the tour earlier (machine busy) had medians of 31-112 ms. So: outdoor scenes reach 40-70 fps, scenes with weather/night/crowds sit near 20 fps here, and the 99th percentile is 100-300 ms in every scene. These numbers are probably 2x pessimistic because of the load, but the **tails are real**: a first visit to a place costs 0.4-1.9 s in a single frame (lazy art: tree sets, dino rigs, boss sets, cave light maps, chunk bakes); in the loaded runs single frames reached 4-6 s. The first frame of the whole game costs 3.2 s (start-up art set, `getArt()` on the first `Game.Update` when the player did not pass the character screen, which also calls it).

## 3. Load times (this machine, quiet moment)

| | 1024 (Lớn) | 2048 (Khổng lồ) |
|---|---|---|
| generate a new world (`server.New`) | 7.0 s (5-9 s) | 17.8 s |
| `world.Save()` | 0.13 s | 0.71 s |
| write save (JSON) | 0.11 s, 11.6 MB | 0.59 s, 37.0 MB |
| read save file | 0.26 s | 0.75 s |
| load a save (regenerates terrain, then entities) | 6.0 s | 18.7 s |
| client: first frame after the server is up | 5.2 s (art 3.2 s + first snapshot) | not measured |
| live Go heap after generation | 57 MB | 202 MB |

Under load these doubled (20 s for 1024 once). Generation and loading run in the background (`app.background`), and the menu keeps drawing its backdrop while busy, so it stays responsive; I did not time its frame rate during loading. The load of a save is as slow as generating a world because the terrain is regenerated from the seed; caching the terrain would be the way to shorten it (not done: a save-format change).

## 4. Old saves

There are **no old save files or fixtures** in the repo; the only older-version tests tweak `Version` on a fresh save. `TestEverySaveVersionLoadsTicksAndSavesAgain` now builds a save for each version 1..28 (and both styles where two existed), strips what that version could not have had (later entity kinds, pets, drops, bosses, stations, difficulty, the slot inventory before 4 as per-item counts), sends it through JSON, loads it, checks the player's items and health come back, steps 200 ticks, saves (must be the current version), JSON, loads again, steps. **All 28 versions pass** (sim package, about 60 s). Limits: these are synthesised, not real old files, so anything an old game wrote in a way the current structs no longer express (renamed JSON fields, since-removed fields) cannot be tested; versions of the style generators before the current ones are loaded through the current generator code, which is exactly what the game does.

## 5. Fixed in this branch

- `voiceMemory.engaged` pruning (tribesound.go).
- Tools and tests listed above. No art or rules were changed.

## Left / recommendations

1. Re-run a 10-minute `soak -mode soak -gc -memprofile` (and the real game) on a quiet machine to confirm or clear the Ebitengine byte-pool growth (finding 1). Highest priority before handing the game to testers; if it is real, bound the number of chunks baked per frame.
2. Do not warm all boss/sea-monster moves ahead (finding 2): 230 MB.
3. The first-sight hitches (0.4-1.9 s) deserve a warm-up spread over frames (draw one set per frame during the loading screen); the 3.2 s start-up art could be drawn behind the "Đang mở thế giới" status instead of freezing the window at the first game frame.
4. Cache the generated terrain in the save (or a sidecar) so loading a 2048 save takes under 2 s instead of 19 s.
5. Memory advice for testers: a 2048 session can reach several GB of working set; recommend 8 GB machines, quality 75 does not reduce it.
6. Zoom +1 frame times and a long walk on foot on 2048 were not measured.
