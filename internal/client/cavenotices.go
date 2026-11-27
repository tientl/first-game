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

// caveDirs are the eight ways, from the north round by the east.
var caveDirs = [8]string{"bắc", "đông bắc", "đông", "đông nam", "nam", "tây nam", "tây", "tây bắc"}

// paintingText is what a picture on a cave wall shows.
func paintingText(arg uint8) string {
	pic, dir := sim.PaintingOf(int(arg))
	switch pic {
	case sim.PaintingHunt:
		return "Hình vẽ cổ: những người đi săn vây quanh một con thú có sừng"
	case sim.PaintingHoard:
		return "Hình vẽ cổ: một chiếc rương, mũi tên chỉ về phía " + caveDirs[dir%8] + " của hang"
	case sim.PaintingStar:
		return "Hình vẽ cổ: một vì sao lửa rơi xuống hòn đảo ở giữa biển"
	}
	return "Hình vẽ cổ: đàn thú cổ dài kéo nhau đi qua các hòn đảo"
}
