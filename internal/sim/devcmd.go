package sim

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// The dev console's commands: a line of text typed by a player in dev mode,
// run against the world (see dev.go). Names of commands, items, beasts and
// places are matched without regard to case or Vietnamese accents.

// DevResult is what a dev command did, to show in the console.
type DevResult struct {
	Lines    []string
	Revealed bool // the player's explored map changed: send it again
}

func (r *DevResult) say(format string, a ...any) {
	r.Lines = append(r.Lines, fmt.Sprintf(format, a...))
}

// DevHelp is the console's list of commands.
var DevHelp = []string{
	"give <đồ> [số]        cho đồ (give go 20, give \"rìu đá\")",
	"tp <nơi>             làng | trùm N | hang | kim tự tháp | spawn | x y",
	"spawn <thú> [số]      gọi thú ra quanh mình (trùm thì không)",
	"time <0-1|day|dusk|night>   season <xuân|hạ|thu|đông>",
	"weather <clear|rain|wind|storm|tornado|snow|auto>",
	"speed <hệ số>         tốc độ đi (1 là thường)",
	"kill [bán kính]       giết thú hoang quanh mình (mặc định 12 ô)",
	"heal   god [on|off]   reveal   pos   boat (thuyền dưới chân)   list items|beasts [lọc]   help",
	"freebuild [on|off]    xây tự do: chế tạo, dựng không tốn gì · B: bảng xây · Shift+chuột phải: gỡ",
}

// devMore are commands that live with what they work on, by canonical name.
var devMore = map[string]func(w *World, r *DevResult, p *Player, args []string){}

// devCommands maps each command's folded names to its canonical one.
var devCommands = map[string]string{
	"give": "give", "cho": "give", "item": "give",
	"tp": "tp", "teleport": "tp", "dichchuyen": "tp", "den": "tp",
	"spawn": "spawn", "goi": "spawn", "tha": "spawn",
	"time": "time", "gio": "time", "thoigian": "time",
	"season": "season", "mua": "season",
	"weather": "weather", "thoitiet": "weather", "troi": "weather",
	"speed": "speed", "tocdo": "speed",
	"kill": "kill", "giet": "kill",
	"heal": "heal", "hoimau": "heal", "chua": "heal",
	"god": "god", "batu": "god",
	"reveal": "reveal", "bando": "reveal", "map": "reveal",
	"help": "help", "trogiup": "help", "?": "help",
	"list": "list", "ds": "list", "danhsach": "list",
	"pos": "pos", "where": "pos", "odau": "pos",
	"boat": "boat", "thuyen": "boat",
	"freebuild": "freebuild", "build": "freebuild", "creative": "freebuild", "fb": "freebuild",
	"xaytudo": "freebuild", "xay": "freebuild",
}

// DevCommand runs one console line for player id.
func (w *World) DevCommand(id PlayerID, line string) DevResult {
	var r DevResult
	p := w.devPlayer(id)
	if p == nil {
		r.say("Không có người chơi")
		return r
	}
	args := devSplit(line)
	if len(args) == 0 {
		return r
	}
	name := fold(args[0])
	if args[0] == "?" {
		name = "?"
	}
	cmd, ok := devCommands[name]
	if !ok {
		r.say("Không có lệnh %q. Gõ help để xem các lệnh.", args[0])
		return r
	}
	args = args[1:]
	if f, ok := devMore[cmd]; ok {
		f(w, &r, p, args) // commands kept with what they work on, e.g. skills.go's fight and skill
		return r
	}
	switch cmd {
	case "help":
		r.Lines = append(r.Lines, DevHelp...)
	case "give":
		w.devGive(&r, id, args)
	case "tp":
		w.devTP(&r, p, args)
	case "spawn":
		w.devSpawnCmd(&r, id, args)
	case "time":
		w.devTimeCmd(&r, args)
	case "season":
		w.devSeasonCmd(&r, args)
	case "weather":
		w.devWeatherCmd(&r, args)
	case "speed":
		f := 1.0
		if len(args) > 0 {
			v, err := strconv.ParseFloat(strings.ReplaceAll(args[0], ",", "."), 64)
			if err != nil {
				r.say("speed cần một số, ví dụ speed 3")
				return r
			}
			f = v
		}
		r.say("Tốc độ ×%g", w.DevSetSpeed(id, f))
	case "kill":
		rad := DevKillRadius
		if len(args) > 0 {
			if v, err := strconv.ParseFloat(args[0], 64); err == nil && v > 0 {
				rad = min(v, 200)
			}
		}
		r.say("Đã giết %d con trong %g ô", w.DevKill(id, rad), rad)
	case "heal":
		w.DevHeal(id)
		r.say("Đầy máu, no bụng")
	case "god":
		on := !p.dev.god
		if len(args) > 0 {
			switch fold(args[0]) {
			case "on", "bat", "1":
				on = true
			case "off", "tat", "0":
				on = false
			}
		}
		w.DevSetGod(id, on)
		if on {
			r.say("Bất tử: bật")
		} else {
			r.say("Bất tử: tắt")
		}
	case "freebuild":
		w.devFreeBuildCmd(&r, p, args)
	case "reveal":
		w.DevReveal(id)
		r.Revealed = true
		r.say("Đã mở toàn bộ bản đồ")
	case "boat":
		switch {
		case p.boat != 0:
			r.say("Đang ở trên thuyền rồi")
		case !w.boatFits(BoatSail, p.X, p.Y):
			r.say("Phải ở giữa nước mới thả thuyền được")
		default:
			w.nextBoat++
			w.boats = append(w.boats, &Boat{ID: w.nextBoat, Kind: BoatSail, X: p.X, Y: p.Y, HP: BoatSail.MaxHP(), Rider: p.ID})
			p.boat, p.swim = w.nextBoat, 0
			r.say("Đã lên %s", BoatSail.Name())
		}
	case "pos":
		r.say("Đang ở %.1f %.1f, ngày %d, %s, %s", p.X, p.Y, DayOf(w.tick), w.Season().Name(), w.Weather().Name())
	case "list":
		w.devList(&r, args)
	}
	return r
}

// devFreeBuildCmd is `freebuild [on|off]`: without a word it turns free
// build over.
func (w *World) devFreeBuildCmd(r *DevResult, p *Player, args []string) {
	for len(args) > 0 && (fold(args[0]) == "tu" || fold(args[0]) == "do") {
		args = args[1:] // "xây tự do bật"
	}
	on := !w.freeBuild(p)
	if len(args) > 0 {
		switch fold(args[0]) {
		case "on", "bat", "1":
			on = true
		case "off", "tat", "0":
			on = false
		default:
			r.say("freebuild on hoặc freebuild off")
			return
		}
	}
	if !w.DevSetFreeBuild(p.ID, on) {
		r.say("Xây tự do chỉ có trong chế độ dev")
		return
	}
	if on {
		r.say("Xây tự do: bật — chế tạo và dựng không tốn gì, chặt và đập một nhát")
		r.say("B: bảng xây (chọn rồi chuột trái đặt, giữ kéo để đặt liền) · Shift + chuột phải: gỡ công trình")
	} else {
		r.say("Xây tự do: tắt")
	}
}

// devSplit splits a line into words; "quoted words" stay together.
func devSplit(line string) []string {
	var out []string
	var cur strings.Builder
	quoted, has := false, false
	for _, c := range line {
		switch {
		case c == '"':
			quoted, has = !quoted, true
		case unicode.IsSpace(c) && !quoted:
			if has {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(c)
			has = true
		}
	}
	if has {
		out = append(out, cur.String())
	}
	return out
}

// nameAndCount splits "thịt nướng 5" into the name and the count; the
// count is def when the last word is not a number.
func nameAndCount(args []string, def int) (string, int) {
	if len(args) >= 2 {
		if n, err := strconv.Atoi(args[len(args)-1]); err == nil {
			return strings.Join(args[:len(args)-1], " "), n
		}
	}
	return strings.Join(args, " "), def
}

func (w *World) devGive(r *DevResult, id PlayerID, args []string) {
	name, n := nameAndCount(args, 1)
	if name == "" {
		r.say("give <đồ> [số]; list items để xem tên đồ")
		return
	}
	k, ok := FindItem(name)
	if !ok {
		r.say("Không biết đồ %q%s", name, suggest(devItemNames(), name))
		return
	}
	n = min(max(n, 1), 999)
	got := w.DevGive(id, k, n)
	if got < n {
		r.say("Đã cho %d %s (túi đầy, thiếu %d)", got, k.Def().Name, n-got)
		return
	}
	r.say("Đã cho %d %s", got, k.Def().Name)
}

func (w *World) devTP(r *DevResult, p *Player, args []string) {
	if len(args) == 0 {
		r.say("tp <làng|trùm N|hang|kim tự tháp|spawn|x y>")
		return
	}
	if len(args) == 2 {
		x, e1 := strconv.ParseFloat(args[0], 64)
		y, e2 := strconv.ParseFloat(args[1], 64)
		if e1 == nil && e2 == nil {
			if w.DevTeleport(p.ID, x, y) {
				r.say("Đã tới %.1f %.1f", x, y)
			} else {
				r.say("%g %g nằm ngoài bản đồ (%d×%d)", x, y, w.width, w.height)
			}
			return
		}
	}
	where := fold(strings.Join(args, ""))
	place, n, label := DevSpawn, 0, "spawn"
	switch {
	case where == "spawn" || where == "nha" || where == "home" || where == "start":
	case where == "village" || where == "lang" || where == "langmac" || where == "tribe":
		place, label = DevVillage, "làng gần nhất"
	case where == "cave" || where == "hang" || where == "hangdong":
		place, label = DevCave, "hang gần nhất"
	case where == "pyramid" || where == "kimtuthap" || where == "kimtu":
		place, label = DevPyramid, "kim tự tháp"
	case strings.HasPrefix(where, "boss") || strings.HasPrefix(where, "trum"):
		place = DevBoss
		rest := strings.TrimPrefix(strings.TrimPrefix(where, "boss"), "trum")
		if v, err := strconv.Atoi(rest); err == nil {
			n = v
		} else {
			for i := range NumBosses {
				if k := BossKind(i); rest != "" && strings.Contains(fold(k.Name()), rest) {
					n = i + 1
					break
				}
			}
		}
		if n < 1 || n > NumBosses {
			var names []string
			for i := range NumBosses {
				names = append(names, fmt.Sprintf("%d %s", i+1, BossKind(i).Name()))
			}
			r.say("tp boss <1-%d>: %s", NumBosses, strings.Join(names, ", "))
			return
		}
		label = "trùm " + BossKind(n-1).Name()
	default:
		r.say("Không biết nơi %q: làng, trùm N, hang, kim tự tháp, spawn hay x y", strings.Join(args, " "))
		return
	}
	x, y, ok := w.DevFind(p.ID, place, n)
	if !ok {
		r.say("Thế giới này không có %s", label)
		return
	}
	w.DevTeleport(p.ID, x, y)
	r.say("Đã tới %s (%.1f %.1f)", label, x, y)
}

func (w *World) devSpawnCmd(r *DevResult, id PlayerID, args []string) {
	name, n := nameAndCount(args, 1)
	if name == "" {
		r.say("spawn <thú> [số]; list beasts để xem tên thú")
		return
	}
	k, ok := FindCreature(name)
	if !ok {
		r.say("Không biết thú %q%s", name, suggest(devCreatureNames(), name))
		return
	}
	if k.Boss() {
		r.say("%s chỉ ở hang ổ của nó: tp boss", k.Name())
		return
	}
	n = min(max(n, 1), 50)
	got := w.DevSpawn(id, k, n)
	if got == 0 {
		r.say("Không có chỗ cho %s quanh đây", k.Name())
		return
	}
	r.say("Đã gọi %d %s", got, k.Name())
}

func (w *World) devTimeCmd(r *DevResult, args []string) {
	if len(args) == 0 {
		r.say("Ngày %d, %.2f của ngày. time <0-1|day|dusk|night>", DayOf(w.tick), DayFraction(w.tick))
		return
	}
	var f float64
	switch fold(strings.Join(args, "")) {
	case "dawn", "binhminh", "sang", "sangsom":
		f = 0.01
	case "day", "ngay", "morning":
		f = 0.1
	case "noon", "trua":
		f = 0.3
	case "dusk", "evening", "chieu", "hoanghon", "changvang":
		f = DuskAt(w.tick) + 0.02
	case "night", "dem", "toi":
		f = NightAt(w.tick) + 0.08
	case "midnight", "nuadem":
		f = 0.9
	default:
		v, err := strconv.ParseFloat(strings.ReplaceAll(args[0], ",", "."), 64)
		if err != nil || v < 0 || v >= 1 {
			r.say("time cần số từ 0 tới dưới 1, hoặc day, dusk, night")
			return
		}
		f = v
	}
	w.DevSetTime(f)
	r.say("Ngày %d, %.2f của ngày", DayOf(w.tick), DayFraction(w.tick))
}

func (w *World) devSeasonCmd(r *DevResult, args []string) {
	s := NumSeasons
	switch fold(strings.Join(args, "")) {
	case "spring", "xuan", "muaxuan":
		s = Spring
	case "summer", "ha", "he", "muaha", "muahe":
		s = Summer
	case "autumn", "fall", "thu", "muathu":
		s = Autumn
	case "winter", "dong", "muadong":
		s = Winter
	}
	if s == NumSeasons {
		r.say("season <xuân|hạ|thu|đông> (spring, summer, autumn, winter); bây giờ là %s", w.Season().Name())
		return
	}
	if !w.DevSetSeason(s) {
		r.say("Đang là %s rồi", s.Name())
		return
	}
	r.say("Sang %s, ngày %d", s.Name(), DayOf(w.tick))
}

func (w *World) devWeatherCmd(r *DevResult, args []string) {
	wt := NumWeathers
	switch fold(strings.Join(args, "")) {
	case "clear", "sun", "sunny", "quang", "troiquang", "nang":
		wt = WeatherClear
	case "rain", "mua", "muarao":
		wt = WeatherRain
	case "wind", "windy", "gio":
		wt = WeatherWind
	case "storm", "thunder", "giong", "bao", "giongbao":
		wt = WeatherStorm
	case "tornado", "loc", "locxoay":
		wt = WeatherTornado
	case "snow", "tuyet", "tuyetroi":
		wt = WeatherSnow
	case "auto", "off", "tat", "tunhien", "reset":
		w.DevClearWeather()
		r.say("Thời tiết theo tự nhiên: %s", w.Weather().Name())
		return
	}
	if wt == NumWeathers {
		r.say("weather <clear|rain|wind|storm|tornado|snow|auto>; bây giờ: %s", w.Weather().Name())
		return
	}
	w.DevForceWeather(wt)
	r.say("Thời tiết: %s (giữ đến khi weather auto)", wt.Name())
}

func (w *World) devList(r *DevResult, args []string) {
	what := ""
	if len(args) > 0 {
		what = fold(args[0])
	}
	filter := fold(strings.Join(args[min(1, len(args)):], ""))
	var names []devName
	switch what {
	case "items", "item", "do", "dodac":
		names = devItemNames()
	case "beasts", "beast", "creatures", "creature", "thu", "animals", "mobs":
		names = devCreatureNames()
	default:
		r.say("list items [lọc] hoặc list beasts [lọc]")
		return
	}
	var cur string
	n := 0
	for _, d := range names {
		if filter != "" && !strings.Contains(d.key, filter) {
			continue
		}
		e := fmt.Sprintf("%d %s", d.id, d.name)
		if cur != "" && len([]rune(cur))+len([]rune(e)) > 90 {
			r.Lines = append(r.Lines, cur)
			cur = ""
		}
		if cur != "" {
			cur += " · "
		}
		cur += e
		n++
	}
	if cur != "" {
		r.Lines = append(r.Lines, cur)
	}
	if n == 0 {
		r.say("Không có tên nào khớp %q", filter)
	}
}

// --- names ------------------------------------------------------------------

// devName is something the console can name: an item or a creature kind.
type devName struct {
	id   int
	name string // as the game shows it
	key  string // folded
}

func devItemNames() []devName {
	var out []devName
	for k := ItemNone + 1; k < NumItems; k++ {
		if n := k.Def().Name; k.Valid() && strings.TrimSpace(n) != "Đầu" {
			out = append(out, devName{int(k), n, fold(n)})
		}
	}
	return out
}

func devCreatureNames() []devName {
	var out []devName
	for k := CreatureNone + 1; k < NumCreatureKinds; k++ {
		if n := k.Name(); n != "" {
			out = append(out, devName{int(k), n, fold(n)})
		}
	}
	return out
}

// devItemAliases are English names for the common items.
var devItemAliases = map[string]ItemKind{
	"wood": ItemWood, "stone": ItemStone, "hay": ItemGrass, "fiber": ItemGrass, "grass": ItemGrass,
	"berry": ItemBerry, "stick": ItemStick, "axe": ItemAxe, "meat": ItemMeat, "fish": ItemFish,
	"torch": ItemTorch, "spear": ItemSpear, "rod": ItemRod, "coal": ItemCoal,
	"hide": ItemHide, "bone": ItemBone, "fur": ItemFur, "saddle": ItemSaddle,
	"potion": ItemPotion, "raft": ItemRaft, "boat": ItemSailboat, "sailboat": ItemSailboat,
	"honey": ItemHoney, "banana": ItemBanana, "battleaxe": ItemBattleAxe,
	"bamboo": ItemBamboo, "gourd": ItemGourd, "pitcherwater": ItemPitcherWater, "flax": ItemFlax, "vine": ItemVine,
	"rope": ItemRope, "poisonmushroom": ItemPoisonMushroom, "nightshade": ItemNightshade, "aloe": ItemAloe,
	"herb": ItemHerb, "moss": ItemMoss, "seaweed": ItemSeaweed, "lily": ItemLily, "lotusseed": ItemLotusSeed, "lotus": ItemLotusSeed,
	"horsetail": ItemHorsetail, "rice": ItemRice, "corn": ItemCorn, "squash": ItemSquash, "squashseed": ItemSquashSeed,
	"toadskin": ItemToadSkin, "poisongland": ItemPoisonGland, "glowgland": ItemGlowGland,
	"tunic": ItemTunic, "mantle": ItemMantle, "legwraps": ItemLegWraps, "boots": ItemBoots,
}

// devBeastAliases are English names for the beasts.
var devBeastAliases = map[string]CreatureKind{
	"rabbit": CreatureRabbit, "snake": CreatureSnake, "worm": CreatureWorm, "fish": CreatureFish,
	"crab": CreatureCrab, "croc": CreatureCroc, "crocodile": CreatureCroc,
	"trike": CreatureTrike, "triceratops": CreatureTrike, "stego": CreatureStego, "stegosaurus": CreatureStego,
	"brachio": CreatureBrachio, "sauropod": CreatureBrachio, "raptor": CreatureRaptor,
	"rex": CreatureRex, "trex": CreatureRex, "ptero": CreaturePtero, "plesio": CreaturePlesio,
	"mammoth": CreatureMammoth, "sabre": CreatureSabre, "sabretooth": CreatureSabre, "smilodon": CreatureSabre,
	"ornitho": CreatureOrnitho, "thesc": CreatureThesc, "toro": CreatureToro, "achero": CreatureAchero,
	"quetzal": CreatureQuetzal, "brachy": CreatureBrachy, "champso": CreatureChampso,
	"spider": CreatureSpider, "bear": CreatureBear, "tribesman": CreatureTribesman, "warrior": CreatureTribesman,
	"mosa": CreatureMosa, "mosasaur": CreatureMosa, "kraken": CreatureKraken, "squid": CreatureKraken,
	"scorpion": CreatureScorpion, "monkey": CreatureMonkey,
	"toad": CreatureToad, "poisontoad": CreatureToad, "greentoad": CreatureToad, "glowtoad": CreatureGlowToad,
}

// FindItem finds the item a player means by name: its number, an English
// alias, or its Vietnamese name, whole or a unique part of it, without
// regard to case or accents.
func FindItem(name string) (ItemKind, bool) {
	if k, ok := devItemAliases[fold(name)]; ok {
		return k, true
	}
	id, ok := devMatch(devItemNames(), name)
	return ItemKind(id), ok
}

// FindCreature finds the creature kind a player means by name, as FindItem.
func FindCreature(name string) (CreatureKind, bool) {
	if k, ok := devBeastAliases[fold(name)]; ok {
		return k, true
	}
	id, ok := devMatch(devCreatureNames(), name)
	return CreatureKind(id), ok
}

// devMatch picks from names: by number, then a whole name, then the one
// name starting with q, then the one name holding q. Several equally good
// matches are no match.
func devMatch(names []devName, q string) (int, bool) {
	if n, err := strconv.Atoi(strings.TrimSpace(q)); err == nil {
		for _, d := range names {
			if d.id == n {
				return n, true
			}
		}
		return 0, false
	}
	fq := fold(q)
	if fq == "" {
		return 0, false
	}
	for _, test := range []func(string) bool{
		func(k string) bool { return k == fq },
		func(k string) bool { return strings.HasPrefix(k, fq) },
		func(k string) bool { return strings.Contains(k, fq) },
	} {
		found, n := 0, 0
		for _, d := range names {
			if test(d.key) {
				found, n = d.id, n+1
			}
		}
		if n == 1 {
			return found, true
		}
		if n > 1 {
			return 0, false
		}
	}
	return 0, false
}

// suggest lists a few names holding q, for an error line.
func suggest(names []devName, q string) string {
	fq := fold(q)
	var hits []string
	for _, d := range names {
		if fq != "" && strings.Contains(d.key, fq) {
			hits = append(hits, d.name)
		}
	}
	if len(hits) == 0 {
		return ""
	}
	if len(hits) > 6 {
		hits = append(hits[:6], "…")
	}
	return ": " + strings.Join(hits, ", ")
}

// foldVN maps each accented Vietnamese letter to its bare letter.
var foldVN = func() map[rune]rune {
	m := map[rune]rune{}
	for _, g := range []string{
		"aàáảãạăằắẳẵặâầấẩẫậ", "eèéẻẽẹêềếểễệ", "iìíỉĩị",
		"oòóỏõọôồốổỗộơờớởỡợ", "uùúủũụưừứửữự", "yỳýỷỹỵ", "dđ",
	} {
		rs := []rune(g)
		for _, c := range rs[1:] {
			m[c] = rs[0]
		}
	}
	return m
}()

// fold lowers s, strips Vietnamese accents and drops everything but letters
// and digits, so "Rìu đá" and "riu da" match.
func fold(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if m, ok := foldVN[c]; ok {
			c = m
		}
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			b.WriteRune(c)
		}
	}
	return b.String()
}
