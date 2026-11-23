package client

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// ExportAtlas writes the game's art and catalogue to dir, for documentation:
// every item icon, every world object and animal sprite as a PNG, and
// catalog.json describing them with the recipes. It must run inside the game
// loop, where images can be created and read.
func ExportAtlas(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	g := New(nil, "")
	g.art = getArt()

	save := func(name string, img *ebiten.Image) (string, error) {
		if img == nil {
			return "", nil
		}
		b := img.Bounds()
		rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		img.ReadPixels(rgba.Pix) // premultiplied, as image.RGBA is
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		defer f.Close()
		return name, png.Encode(f, rgba)
	}

	type itemOut struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Info   string  `json:"info"`
		Hunger float64 `json:"hunger"`
		Health float64 `json:"health"`
		Tool   bool    `json:"tool"`
		Stack  int     `json:"stack"`
		Icon   string  `json:"icon"`
	}
	type ingOut struct {
		Item  int `json:"item"`
		Count int `json:"count"`
	}
	type recipeOut struct {
		Name        string   `json:"name"`
		Inputs      []ingOut `json:"inputs"`
		Gives       int      `json:"gives,omitempty"`
		Places      int      `json:"places,omitempty"`
		NearFire    bool     `json:"nearFire,omitempty"`
		NearFurnace bool     `json:"nearFurnace,omitempty"`
	}
	type spriteOut struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Sprite string `json:"sprite"`
		Big    string `json:"big,omitempty"`
	}
	var out struct {
		Items     []itemOut   `json:"items"`
		Recipes   []recipeOut `json:"recipes"`
		Entities  []spriteOut `json:"entities"`
		Creatures []spriteOut `json:"creatures"`
	}

	for k := sim.ItemKind(1); k < sim.NumItems; k++ {
		icon, err := save(fmt.Sprintf("item_%d.png", k), g.art.icons[k])
		if err != nil {
			return err
		}
		d := k.Def()
		out.Items = append(out.Items, itemOut{int(k), d.Name, itemInfo[k], d.Hunger, d.Health, d.Tool, k.MaxCount(), icon})
	}
	for id := sim.RecipeID(0); id < sim.NumRecipes; id++ {
		r, ok := sim.RecipeByID(id)
		if !ok || r.Name == "" {
			continue
		}
		ro := recipeOut{Name: r.Name, Gives: int(r.Gives), Places: int(r.Places), NearFire: r.NearFire, NearFurnace: r.NearFurnace}
		for _, in := range r.Inputs {
			ro.Inputs = append(ro.Inputs, ingOut{int(in.Item), in.Count})
		}
		out.Recipes = append(out.Recipes, ro)
	}
	for k := sim.EntityKind(1); k < sim.NumEntityKinds; k++ {
		e := protocol.EntityView{ID: sim.EntityID(k), Kind: k, TX: 10, TY: 10, Ready: true, Fuel: sim.FireStartFuel, Stage: sim.CropStages}
		s, _ := g.entitySprite(e)
		so := spriteOut{ID: int(k), Name: k.Name()}
		var err error
		if so.Sprite, err = save(fmt.Sprintf("entity_%d.png", k), s.img); err != nil {
			return err
		}
		if k.Sized() {
			e.Big = true
			b, _ := g.entitySprite(e)
			if so.Big, err = save(fmt.Sprintf("entity_%d_big.png", k), b.img); err != nil {
				return err
			}
		}
		out.Entities = append(out.Entities, so)
	}
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		s, _ := g.creatureSprite(protocol.CreatureView{ID: sim.CreatureID(k), Kind: k}, false)
		name, err := save(fmt.Sprintf("creature_%d.png", k), s.img)
		if err != nil {
			return err
		}
		out.Creatures = append(out.Creatures, spriteOut{ID: int(k), Name: k.Name(), Sprite: name})
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "catalog.json"), b, 0o644)
}
