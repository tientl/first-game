package sim

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

// TestDumpSkillTable writes $SKILL_TABLE: every beast's every move as a
// Markdown table, with the duel table under it (the numbers as they are).
func TestDumpSkillTable(t *testing.T) {
	path := os.Getenv("SKILL_TABLE")
	if path == "" {
		t.Skip("set SKILL_TABLE to a .md path")
	}
	sec := func(ticks int) string { return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", float64(ticks)/TicksPerSecond), "0"), ".") + " s" }
	class := map[SkillClass]string{SkillLight: "nhẹ", SkillHeavy: "mạnh", SkillSpecial: "đặc biệt", SkillCry: "kêu/doạ"}
	shape := func(d *SkillDef) string {
		switch d.Shape {
		case ZoneCone:
			side := "trước mặt"
			if d.Behind {
				side = "sau lưng"
			}
			return fmt.Sprintf("quạt %s %.1f ô, %.0f°", side, d.Reach, 2*d.Width*180/math.Pi)
		case ZoneCircle:
			at := "quanh nó"
			switch d.Anchor {
			case AtAhead:
				at = fmt.Sprintf("trước nó %.1f ô", d.Offset)
			case AtBehind:
				at = fmt.Sprintf("sau nó %.1f ô", d.Offset)
			case AtTarget:
				at = "chỗ đáp"
			}
			if d.Motion == MoveHover {
				at = "dưới nó"
			}
			return fmt.Sprintf("tròn r %.1f ô %s", d.Reach, at)
		case ZoneLine:
			return fmt.Sprintf("đường %.1f ô, rộng %.1f", d.Reach, 2*d.Width)
		}
		return "—"
	}
	fx := func(d *SkillDef) string {
		var out []string
		if d.Hits > 1 {
			out = append(out, fmt.Sprintf("%d đòn", d.Hits))
		}
		if d.Knock > 0 {
			out = append(out, fmt.Sprintf("hất %.1f ô", d.Knock))
		}
		if d.Stun > 0 {
			out = append(out, "choáng "+sec(d.Stun))
		}
		if d.Bleed > 0 {
			out = append(out, fmt.Sprintf("chảy máu %s (%.0f máu)", sec(d.Bleed), BleedPerTick*float64(d.Bleed)))
		}
		if d.Slow > 0 {
			out = append(out, fmt.Sprintf("chậm %s (×%.2f)", sec(d.Slow), SlowShare))
		}
		names := map[Effect]string{FxPin: fmt.Sprintf("đè xuống, cắn %.0f mỗi %s tới %s, bấm %d lần để thoát", PinBite, sec(PinBiteEvery), sec(PinHold), PinMash),
			FxSteal: "cướp đồ ăn", FxCall: "gọi đàn/bầy", FxFlank: "cả bầy vây hai bên", FxScare: "thú nhỏ bỏ chạy",
			FxWarn: "chỉ doạ: không lùi thì nó húc", FxTurnTail: "quay đuôi về phía người chơi", FxTurtle: fmt.Sprintf("đòn đánh vào chỉ còn %.0f%%", TurtleShare*100)}
		for _, e := range []Effect{FxPin, FxSteal, FxCall, FxFlank, FxScare, FxWarn, FxTurnTail, FxTurtle} {
			if d.Fx&e != 0 {
				out = append(out, names[e])
			}
		}
		switch d.Motion {
		case MoveDash:
			out = append(out, fmt.Sprintf("lao %.0f ô/s", d.Speed*TicksPerSecond))
		case MoveLeap:
			out = append(out, fmt.Sprintf("nhảy tới %.1f ô", d.MaxD))
		case MoveHopBack:
			out = append(out, fmt.Sprintf("nhảy lùi %.1f ô", d.Back))
		case MoveSwim:
			out = append(out, "lặn tới gần")
		}
		if d.Torch > 0 {
			out = append(out, "lấy đà lâu hơn "+sec(d.Torch)+" nếu người chơi cầm đuốc")
		}
		if len(out) == 0 {
			return "—"
		}
		return strings.Join(out, "; ")
	}
	when := func(d *SkillDef) string {
		var out []string
		if d.MinD > 0 {
			out = append(out, fmt.Sprintf("cách %.1f–%.1f ô", d.MinD, d.MaxD))
		} else if d.MaxD < 50 {
			out = append(out, fmt.Sprintf("trong %.1f ô", d.MaxD))
		}
		if d.HPBelow > 0 {
			out = append(out, fmt.Sprintf("máu ≤ %.0f%%", d.HPBelow*100))
		}
		needs := map[Need]string{NeedLeader: "con đầu bầy", NeedFood: "người chơi cầm/để đồ ăn gần", NeedWet: "người chơi dưới nước/trên thuyền",
			NeedUp: "đang ngoi lên", NeedHit: "vừa bị đánh lúc đang yên", NeedCalm: "chưa bị chọc giận", NeedAngry: "đang giận", NeedFacing: "người chơi ở phía đầu"}
		if s, ok := needs[d.Need]; ok {
			out = append(out, s)
		}
		if d.Stagger {
			out = append(out, "đòn mạnh làm khựng")
		}
		return strings.Join(out, "; ")
	}
	var b strings.Builder
	b.WriteString("# Bộ chiêu của thú (sim/skills.go)\n\n")
	b.WriteString("20 tick = 1 giây. Sát thương chỉ tính lúc ra đòn (hết lấy đà), cho ai còn đứng trong vùng. ")
	b.WriteString(fmt.Sprintf("Đòn ≥ %d làm khựng thú nhỏ đang lấy đà (khựng %s); đòn bất kỳ cắt tiếng kêu. ", StaggerDamage, sec(StaggerTicks)))
	b.WriteString(fmt.Sprintf("Choáng không nối tiếp: sau khi hết choáng có %s miễn choáng. Bầy thay phiên: Dakotaraptor 1 con, Acheroraptor và Troodon 2 con cùng lúc.\n\n", sec(StunGrace)))
	for _, k := range []CreatureKind{CreatureRex, CreatureRaptor, CreatureAchero, CreatureTroodon, CreaturePtero, CreatureQuetzal,
		CreaturePlesio, CreatureSabre, CreatureTrike, CreatureToro, CreatureStego, CreatureBrachio, CreatureAnkylo, CreatureAnzu,
		CreatureThesc, CreatureOrnitho, CreatureMammoth} {
		b.WriteString(fmt.Sprintf("## %s (%d máu)\n\n", k.Name(), k.MaxHP()))
		b.WriteString("| Chiêu | Loại | Lấy đà | Ra đòn | Hồi | Vùng | Sát thương | Hiệu ứng | Hồi chiêu | Khi nào |\n")
		b.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
		for _, id := range SkillsOf(k) {
			d := Skill(id)
			zone := shape(d)
			if d.Zone {
				zone += " (vùng đỏ)"
			}
			dmg := "—"
			if d.Damage > 0 {
				dmg = fmt.Sprintf("%.0f", d.Damage)
			}
			b.WriteString(fmt.Sprintf("| %s (`%s`) | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n", d.Label, d.Name, class[d.Class],
				sec(d.Windup), sec(d.Active), sec(d.Recover), zone, dmg, fx(d), sec(d.Cooldown), when(d)))
		}
		b.WriteString("\n")
	}
	// The duel table: a player toe to toe with each beast, holding Action.
	kinds := []CreatureKind{CreatureRaptor, CreatureAchero, CreatureSabre, CreatureAnzu, CreatureTrike, CreatureToro, CreatureStego,
		CreatureMammoth, CreatureQuetzal, CreatureRex}
	type kit struct {
		name   string
		weapon ItemKind
		armor  Tier
	}
	kits := []kit{{"rìu đá", ItemAxe, 0}, {"giáo", ItemSpear, 0}, {"kiếm đồng", SwordItem(TierCopper), 0},
		{"kiếm sắt+giáp", SwordItem(TierIron), TierIron}, {"kiếm bạch kim+giáp", SwordItem(TierPlatinum), TierPlatinum},
		{"kiếm kim cương+giáp", SwordItem(TierDiamond), TierDiamond}}
	b.WriteString("## Bảng đấu tay đôi (người chơi đứng yên đánh liên tục, 5 lần)\n\n| Thú |")
	for _, k := range kits {
		b.WriteString(" " + k.name + " |")
	}
	b.WriteString("\n|---|" + strings.Repeat("---|", len(kits)) + "\n")
	for _, kind := range kinds {
		b.WriteString("| " + kind.Name() + " |")
		for _, k := range kits {
			wins, secs, hp := 0, 0.0, 0.0
			for s := range 5 {
				if r := duel(kind, k.weapon, k.armor, s); r.won {
					wins++
					secs += r.secs
					hp += r.hpLeft
				}
			}
			cell := fmt.Sprintf("%d/5", wins)
			if wins > 0 {
				cell += fmt.Sprintf(", %.1f s, còn %.0f máu", secs/float64(wins), hp/float64(wins))
			}
			b.WriteString(" " + cell + " |")
		}
		b.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
