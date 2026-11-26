package client

import "survivalgame/internal/sim"

// caveNoticeText is the text of a notice of the natural caves.
func caveNoticeText(kind sim.NoticeKind, arg uint8) (string, bool) {
	switch kind {
	case sim.NoticeDeepCaves:
		return "Xuống tầng hang sâu — tối hơn, nhiều ngọc và nhiều hiểm nguy hơn", true
	case sim.NoticeUpperCaves:
		return "Leo dây lên tầng hang trên", true
	case sim.NoticeLavaBurn:
		return "Bỏng! Đừng đứng sát dòng dung nham", true
	case sim.NoticePainting:
		return paintingText(arg), true
	case sim.NoticeFossil:
		return "Đào được " + sim.ItemKind(arg).Def().Name + " từ bộ xương hoá thạch", true
	}
	return "", false
}

// paintingText is what a picture on a cave wall shows.
func paintingText(arg uint8) string {
	return "Hình vẽ cổ trên vách đá"
}
