package client

// The credits ("Ghi công"): everything the game owes to others, in one table.
// The screen (creditsui.go) draws it, and credits_test.go checks that every
// recording in CREDITS.md and assets/voices/SOURCES.md and every Go module the
// game links is in it, so a new sound or library cannot be forgotten. The
// licences themselves are in LICENSES.txt at the repository root (made by
// `go run ./cmd/licenses`).

// creditAuthor is the line under the game's name. The owner fills it in.
const creditAuthor = "Tác giả: (điền tên)"

// creditLicenceFile tells where the full licences are.
const creditLicenceFile = "Toàn văn các giấy phép nằm trong file LICENSES.txt cạnh game."

// creditEntry is one thing credited: its name, who made it, its licence and
// where it comes from. Any may be empty.
type creditEntry struct {
	Name, By, Licence, URL string
}

// creditSection is a heading and what is credited under it.
type creditSection struct {
	Title   string
	Intro   string
	Entries []creditEntry
}

// soundCredit is one recording the creatures' voices are made from, as in
// assets/voices/SOURCES.md: its file, what it is, its author, its licence and
// its page. Author and licence are as that file gives them.
type soundCredit struct {
	File, What, Author, Licence, URL string
}

// creditSounds are the recordings, in the order of SOURCES.md. Those under
// CC BY 4.0 must be credited here; the others are by courtesy.
var creditSounds = []soundCredit{
	{"nri-BisonYELL.mp3", "American bison bulls bellowing, Yellowstone", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-bison.htm"},
	{"yell-YELLMM8K2005914Bison.mp3", "bison bulls bellowing in the rut, Mary Mountain Trail", "NPS / Shan Burson", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-bison.htm"},
	{"yell-YELLMM8K2005918Bison.mp3", "bison bellowing and grunting", "NPS / Shan Burson", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-bison.htm"},
	{"yell-ElkBugle1.mp3", "a bull elk bugling", "NPS / Neal Herbert", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm"},
	{"yell-ElkBugle2.mp3", "a bull elk bugling", "NPS / Neal Herbert", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm"},
	{"yell-YELLElkBullandCows20151018T012.mp3", "a bull elk bugling to his cows", "NPS / Jennifer Jerrett", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm"},
	{"yell-YELLElkChorus150922T04ms01.mp3", "bull elk bugling at night", "NPS / Jennifer Jerrett", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm"},
	{"nri-ElkBuglingGRSA.mp3", "a bull elk bugling, Great Sand Dunes", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-elk.htm"},
	{"yell-YELLMJ23ElkCalf20051116.mp3", "an elk calf calling", "NPS / Shan Burson", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm"},
	{"yell-YELLGrizzlyBearVocalizations20160520Android1.mp3", "a grizzly bear roaring and moaning", "NPS / Dan Bergum", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/grizzlysounds.htm"},
	{"nri-BearwithCubsDENA.mp3", "a bear with cubs, the cubs bawling, Denali", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-bearcubs.htm"},
	{"nri-AlligatorEVER1.mp3", "American alligators bellowing, Everglades", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-alligator.htm"},
	{"nri-CommonRavenGRTE.mp3", "a common raven croaking, Grand Teton", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-common-raven.htm"},
	{"yell-RavenBlackSandBasin.mp3", "common ravens calling", "NPS / David Restivo", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-raven.htm"},
	{"nri-OspreyGRTE.mp3", "an osprey calling, Grand Teton", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-osprey.htm"},
	{"nri-BaldEagleYELL.mp3", "a bald eagle calling, Yellowstone", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-bald-eagle.htm"},
	{"nri-PeregrineandRaven.mp3", "a peregrine falcon cacking and a raven", "NPS / Bryce Canyon National Park", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-peregrine-raven_bryce.htm"},
	{"yell-YELLFLBCSACR20075171.mp3", "sandhill cranes calling", "NPS / Shan Burson", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-sandhillcrane.htm"},
	{"nri-SpottedOwlYOSE.mp3", "a spotted owl hooting, Yosemite", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-spotted-owl.htm"},
	{"nri-PigFrogEvergladesNP.mp3", "a pig frog grunting, Everglades", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-pigfrog.htm"},
	{"nri-AnhingaEVER.mp3", "an anhinga's clicking rattle, Everglades", "NPS", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/subjects/sound/sounds-anhinga.htm"},
	{"yell-YELLMJ23200837redfox.mp3", "a red fox barking and screaming", "NPS / Shan Burson", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://www.nps.gov/yell/learn/photosmultimedia/sounds-redfox.htm"},
	{"Phvi-roar-NOAA-PAGroup-01-harbor-seal-clip.wav", "a male harbor seal roaring under water", "NOAA Northeast Fisheries Science Center, Passive Acoustics Branch", "Public domain (work of the U.S. National Oceanic and Atmospheric Administration, 17 U.S.C. § 105)", "https://www.fisheries.noaa.gov/national/science-data/sounds-ocean-mammals"},
	{"Lion_raring-sound1TamilNadu178.ogg", "a lion roaring", "த*உழவன் (Info-farmer), via Wikimedia Commons", "Public domain (as marked on Wikimedia Commons)", "https://commons.wikimedia.org/wiki/File:Lion_raring-sound1TamilNadu178.ogg"},
	{"Australasian-bittern.wav", "an Australasian bittern booming", "Department of Conservation (NZ), via Wikimedia Commons", "CC BY 4.0", "https://commons.wikimedia.org/wiki/File:Australasian-bittern.wav"},
	{"Elephant_voice_-_trumpeting.ogg", "an elephant trumpeting", "த*உழவன் (Info-farmer), via Wikimedia Commons", "CC0 1.0 (public domain dedication)", "https://commons.wikimedia.org/wiki/File:Elephant_voice_-_trumpeting.ogg"},
	{"161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", "dolphins screaming under water, Caribbean Sea", "Felix Blume (Freesound sound 161691), via Wikimedia Commons", "CC0 1.0 (public domain dedication)", "https://commons.wikimedia.org/wiki/File:161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav"},
	{"157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav", "a donkey braying, south of France", "Felix Blume (Freesound sound 157763), via Wikimedia Commons", "CC0 1.0 (public domain dedication)", "https://commons.wikimedia.org/wiki/File:157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav"},
	{"439280_schots_angry-tiger.wav", "a caged tiger growling and snarling", "schots (Freesound sound 439280), via Wikimedia Commons", "CC0 1.0 (public domain dedication)", "https://commons.wikimedia.org/wiki/File:439280_schots_angry-tiger.wav"},
	{"Pavo_cristatus_(call).ogg", "a peacock calling, Denver Zoo", "Ke4roh, via Wikimedia Commons", "Public domain (as marked on Wikimedia Commons)", "https://commons.wikimedia.org/wiki/File:Pavo_cristatus_(call).ogg"},
	{"Arctocephalus_forsteri_-_sound.ogg", "New Zealand fur seals growling, Stephens Island", "Department of Conservation (NZ), via Wikimedia Commons", "CC BY 4.0", "https://commons.wikimedia.org/wiki/File:Arctocephalus_forsteri_-_sound.ogg"},
	{"Geese_Honking_(loud).ogg", "Canada geese honking", "U.S. Fish and Wildlife Service, via Wikimedia Commons", "Public domain (as marked on Wikimedia Commons)", "https://commons.wikimedia.org/wiki/File:Geese_Honking_(loud).ogg"},
	{"Great_Blue_Heron.ogg", "a great blue heron croaking", "U.S. National Park Service (Wind Cave National Park), via Wikimedia Commons", "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)", "https://commons.wikimedia.org/wiki/File:Great_Blue_Heron.ogg"},
}

// creditSections is everything the screen shows, top to bottom.
func creditSections() []creditSection {
	secs := []creditSection{
		{Title: "Game", Entries: []creditEntry{
			{Name: "Dino Isles", By: creditAuthor},
		}},
		{Title: "Âm thanh",
			Intro: "Tiếng kêu của các sinh vật được làm từ những bản thu âm có thật (đã cắt, lọc sạch, hạ giọng và trộn). " +
				"Các bản dưới CC BY 4.0 được ghi công ngay dưới đây; các bản còn lại thuộc phạm vi công cộng hoặc CC0."},
		{Title: "Phông chữ", Entries: []creditEntry{
			{Name: "Nunito", By: "Vernon Adams, Cyreal, Jacques Le Bailly (Copyright 2014 The Nunito Project Authors)",
				Licence: "SIL Open Font License 1.1", URL: "https://github.com/googlefonts/nunito"},
			{Name: "Go Bold (các mũi tên → ↔)", By: "Bigelow & Holmes Inc.",
				Licence: "BSD 3-Clause", URL: "https://go.dev/blog/go-fonts"},
		}},
		{Title: "Thư viện", Entries: creditLibraries},
		{Title: "Ý tưởng và mã tham khảo",
			Intro: "Hiệu ứng chiến đấu và lửa (tia lửa, vết chém, vết vuốt, gươm, giáo, rìu, nước bắn, " +
				"ngọn lửa trại, đuốc) làm theo công thức của pixelfx, vẽ lại bằng Go.",
			Entries: []creditEntry{
				{Name: "pixelfx", By: "Patch (Copyright 2026 Patch); một phần chuyển thể từ hiệu ứng pixel của Sprite Fusion, Hugo Duprez (Copyright 2026 Hugo Duprez)",
					Licence: "MIT", URL: "https://github.com/patch-gg/pixelfx"},
			}},
		{Title: "Ngôn ngữ lập trình", Entries: []creditEntry{
			{Name: "Go", By: "The Go Authors", Licence: "BSD 3-Clause", URL: "https://go.dev"},
		}},
	}
	// The sounds go under their heading, CC BY first.
	for _, ccby := range []bool{true, false} {
		for _, s := range creditSounds {
			if (s.Licence == "CC BY 4.0") != ccby {
				continue
			}
			secs[1].Entries = append(secs[1].Entries, creditEntry{
				Name: s.What + " (" + s.File + ")", By: s.Author, Licence: s.Licence, URL: s.URL})
		}
	}
	return secs
}

// creditLibraries are the Go modules the game binary links (see
// `go list -deps ./cmd/game`); a test keeps the list whole.
var creditLibraries = []creditEntry{
	{Name: "Ebitengine (github.com/hajimehoshi/ebiten/v2)", By: "Hajime Hashimoto và những người đóng góp", Licence: "Apache License 2.0", URL: "https://ebitengine.org"},
	{Name: "github.com/ebitengine/oto/v3", By: "Hajime Hashimoto và những người đóng góp", Licence: "Apache License 2.0", URL: "https://github.com/ebitengine/oto"},
	{Name: "github.com/ebitengine/purego", By: "The Ebitengine Authors", Licence: "Apache License 2.0", URL: "https://github.com/ebitengine/purego"},
	{Name: "github.com/ebitengine/hideconsole", By: "The Ebitengine Authors", Licence: "Apache License 2.0", URL: "https://github.com/ebitengine/hideconsole"},
	{Name: "github.com/go-text/typesetting", By: "The go-text authors", Licence: "Unlicense hoặc BSD 3-Clause", URL: "https://github.com/go-text/typesetting"},
	{Name: "github.com/srwiley/rasterx", By: "Steven R. Wiley", Licence: "BSD 3-Clause", URL: "https://github.com/srwiley/rasterx"},
	{Name: "golang.org/x/image", By: "The Go Authors", Licence: "BSD 3-Clause", URL: "https://golang.org/x/image"},
	{Name: "golang.org/x/sync", By: "The Go Authors", Licence: "BSD 3-Clause", URL: "https://golang.org/x/sync"},
	{Name: "golang.org/x/sys", By: "The Go Authors", Licence: "BSD 3-Clause", URL: "https://golang.org/x/sys"},
	{Name: "golang.org/x/text", By: "The Go Authors", Licence: "BSD 3-Clause", URL: "https://golang.org/x/text"},
	{Name: "Đi kèm Ebitengine: GLFW, go-gl/gl, go-gl/glfw, oksvg", By: "xem NOTICE trong LICENSES.txt", Licence: "zlib/libpng, BSD 3-Clause, MIT"},
}
