# Credits

## Sounds

The creatures' voices are made from recordings of real animals (see
[assets/voices/SOURCES.md](assets/voices/SOURCES.md) for every recording, its licence and its use).

These recordings are used under the Creative Commons Attribution licence, and changed
(cut, cleaned, slowed, filtered and mixed with others):

- "an Australasian bittern booming" (Australasian-bittern.wav), by Department of Conservation (NZ), CC BY 4.0 (<https://creativecommons.org/licenses/by/4.0/>), <https://commons.wikimedia.org/wiki/File:Australasian-bittern.wav>
- "New Zealand fur seals growling, Stephens Island" (Arctocephalus_forsteri_-_sound.ogg), by Department of Conservation (NZ), CC BY 4.0 (<https://creativecommons.org/licenses/by/4.0/>), <https://commons.wikimedia.org/wiki/File:Arctocephalus_forsteri_-_sound.ogg>

With thanks for the public-domain and CC0 recordings to the U.S. National Park Service,
the U.S. National Oceanic and Atmospheric Administration, the U.S. Fish and Wildlife Service,
and to த*உழவன் (Info-farmer), Felix Blume, schots, Ke4roh on Wikimedia Commons.

## Fonts

The interface is set in **Nunito**, by Vernon Adams, Cyreal and Jacques Le Bailly
(Copyright 2014 The Nunito Project Authors, <https://github.com/googlefonts/nunito>),
used under the SIL Open Font License, Version 1.1. The variable font is embedded unchanged as
[internal/client/fonts/Nunito-Variable.ttf](internal/client/fonts/Nunito-Variable.ttf) (renamed
from `Nunito[wght].ttf`), with the licence beside it in
[internal/client/fonts/OFL.txt](internal/client/fonts/OFL.txt).

The few signs Nunito has not got (arrows such as → and ↔) are drawn from **Go Bold**, one of
the Go fonts by Bigelow & Holmes Inc. (<https://go.dev/blog/go-fonts>), under the BSD 3-Clause
licence of the Go project; it comes with the `golang.org/x/image` module and is not copied into
the repository.

## Effects

The combat effects and the fire (hit sparks, critical hits, the flash and lines of a blow,
claw marks, the sword's slash, the thrust, the spear's sweep, the war axe's cleave, the
splash, blocks and parries, the flames of campfires, torches and bonfires) follow the recipes
of **pixelfx** by Patch (<https://github.com/patch-gg/pixelfx>), whose effects are in part
adapted from Sprite Fusion's pixel-art effects by Hugo Duprez: their timings, shapes and
palettes, redrawn in Go in [internal/client/fx.go](internal/client/fx.go) and
[internal/client/fireart.go](internal/client/fireart.go). Used under the MIT License
(Copyright 2026 Patch, Copyright 2026 Hugo Duprez); the licence is given in full in
[internal/client/LICENSE-pixelfx.txt](internal/client/LICENSE-pixelfx.txt).
