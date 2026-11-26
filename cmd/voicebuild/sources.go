package main

// Where every recording comes from. Only public-domain (US government),
// CC0 and CC BY recordings are used: they may be sold inside a game. The
// raw downloads are not kept in the repository; "voicebuild trim" cuts the
// parts used (clips, below) out of them into assets/voices/src, and
// "voicebuild sources" writes this table out as assets/voices/SOURCES.md.

// source is one downloaded recording.
type source struct {
	file    string // its name in the raw download folder
	url     string // the file itself
	page    string // the page that offers it and states its licence
	author  string
	licence string
	fetched string // the date it was downloaded
	what    string // what is in it
}

const (
	pdNPS  = "Public domain (work of the U.S. National Park Service, 17 U.S.C. § 105)"
	pdNOAA = "Public domain (work of the U.S. National Oceanic and Atmospheric Administration, 17 U.S.C. § 105)"
	pdWiki = "Public domain (as marked on Wikimedia Commons)"
	cc0    = "CC0 1.0 (public domain dedication)"
	ccBY3  = "CC BY 3.0"
	ccBY4  = "CC BY 4.0"
)

var sources = []source{
	// The U.S. National Park Service's sound galleries.
	{"nps/nri-BisonYELL.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-BisonYELL.mp3",
		"https://www.nps.gov/subjects/sound/sounds-bison.htm", "NPS", pdNPS, "2026-11-26", "American bison bulls bellowing, Yellowstone"},
	{"nps/yell-YELLMM8K2005914Bison.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLMM8K2005914Bison.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-bison.htm", "NPS / Shan Burson", pdNPS, "2026-11-26", "bison bulls bellowing in the rut, Mary Mountain Trail"},
	{"nps/yell-YELLMM8K2005918Bison.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLMM8K2005918Bison.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-bison.htm", "NPS / Shan Burson", pdNPS, "2026-11-26", "bison bellowing and grunting"},
	{"nps/yell-ElkBugle1.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-ElkBugle1.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm", "NPS / Neal Herbert", pdNPS, "2026-11-26", "a bull elk bugling"},
	{"nps/yell-ElkBugle2.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-ElkBugle2.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm", "NPS / Neal Herbert", pdNPS, "2026-11-26", "a bull elk bugling"},
	{"nps/yell-YELLElkBullandCows20151018T012.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLElkBullandCows20151018T012.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm", "NPS / Jennifer Jerrett", pdNPS, "2026-11-26", "a bull elk bugling to his cows"},
	{"nps/yell-YELLElkChorus150922T04ms01.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLElkChorus150922T04ms01.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm", "NPS / Jennifer Jerrett", pdNPS, "2026-11-26", "bull elk bugling at night"},
	{"nps/nri-ElkBuglingGRSA.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-ElkBuglingGRSA.mp3",
		"https://www.nps.gov/subjects/sound/sounds-elk.htm", "NPS", pdNPS, "2026-11-26", "a bull elk bugling, Great Sand Dunes"},
	{"nps/yell-YELLMJ23ElkCalf20051116.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLMJ23ElkCalf20051116.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-elk.htm", "NPS / Shan Burson", pdNPS, "2026-11-26", "an elk calf calling"},
	{"nps/yell-YELLGrizzlyBearVocalizations20160520Android1.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLGrizzlyBearVocalizations20160520Android1.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/grizzlysounds.htm", "NPS / Dan Bergum", pdNPS, "2026-11-26", "a grizzly bear roaring and moaning"},
	{"nps/nri-BearwithCubsDENA.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-BearwithCubsDENA.mp3",
		"https://www.nps.gov/subjects/sound/sounds-bearcubs.htm", "NPS", pdNPS, "2026-11-26", "a bear with cubs, the cubs bawling, Denali"},
	{"nps/nri-AlligatorEVER1.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-AlligatorEVER1.mp3",
		"https://www.nps.gov/subjects/sound/sounds-alligator.htm", "NPS", pdNPS, "2026-11-26", "American alligators bellowing, Everglades"},
	{"nps/nri-CommonRavenGRTE.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-CommonRavenGRTE.mp3",
		"https://www.nps.gov/subjects/sound/sounds-common-raven.htm", "NPS", pdNPS, "2026-11-26", "a common raven croaking, Grand Teton"},
	{"nps/yell-RavenBlackSandBasin.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-RavenBlackSandBasin.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-raven.htm", "NPS / David Restivo", pdNPS, "2026-11-26", "common ravens calling"},
	{"nps/nri-OspreyGRTE.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-OspreyGRTE.mp3",
		"https://www.nps.gov/subjects/sound/sounds-osprey.htm", "NPS", pdNPS, "2026-11-26", "an osprey calling, Grand Teton"},
	{"nps/nri-BaldEagleYELL.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-BaldEagleYELL.mp3",
		"https://www.nps.gov/subjects/sound/sounds-bald-eagle.htm", "NPS", pdNPS, "2026-11-26", "a bald eagle calling, Yellowstone"},
	{"nps/nri-PeregrineandRaven.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-PeregrineandRaven.mp3",
		"https://www.nps.gov/subjects/sound/sounds-peregrine-raven_bryce.htm", "NPS / Bryce Canyon National Park", pdNPS, "2026-11-26", "a peregrine falcon cacking and a raven"},
	{"nps/yell-YELLFLBCSACR20075171.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLFLBCSACR20075171.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-sandhillcrane.htm", "NPS / Shan Burson", pdNPS, "2026-11-26", "sandhill cranes calling"},
	{"nps/nri-SpottedOwlYOSE.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-SpottedOwlYOSE.mp3",
		"https://www.nps.gov/subjects/sound/sounds-spotted-owl.htm", "NPS", pdNPS, "2026-11-26", "a spotted owl hooting, Yosemite"},
	{"nps/nri-PigFrogEvergladesNP.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-PigFrogEvergladesNP.mp3",
		"https://www.nps.gov/subjects/sound/sounds-pigfrog.htm", "NPS", pdNPS, "2026-11-26", "a pig frog grunting, Everglades"},
	{"nps/nri-AnhingaEVER.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/nri/avElement/nri-AnhingaEVER.mp3",
		"https://www.nps.gov/subjects/sound/sounds-anhinga.htm", "NPS", pdNPS, "2026-11-26", "an anhinga's clicking rattle, Everglades"},
	{"nps/yell-YELLMJ23200837redfox.mp3", "https://www.nps.gov/nps-audiovideo/legacy/mp3/imr/avElement/yell-YELLMJ23200837redfox.mp3",
		"https://www.nps.gov/yell/learn/photosmultimedia/sounds-redfox.htm", "NPS / Shan Burson", pdNPS, "2026-11-26", "a red fox barking and screaming"},

	// NOAA's Northeast Fisheries Science Center, Passive Acoustics Branch.
	{"noaa/Phvi-roar-NOAA-PAGroup-01-harbor-seal-clip.wav", "https://www.fisheries.noaa.gov/s3/2023-06/Phvi-roar-NOAA-PAGroup-01-harbor-seal-clip.wav",
		"https://www.fisheries.noaa.gov/national/science-data/sounds-ocean-mammals", "NOAA Northeast Fisheries Science Center, Passive Acoustics Branch", pdNOAA, "2026-11-27", "a male harbor seal roaring under water"},

	// Wikimedia Commons.
	{"commons/Lion_raring-sound1TamilNadu178.ogg", "https://upload.wikimedia.org/wikipedia/commons/7/7d/Lion_raring-sound1TamilNadu178.ogg",
		"https://commons.wikimedia.org/wiki/File:Lion_raring-sound1TamilNadu178.ogg", "த*உழவன் (Info-farmer), via Wikimedia Commons", pdWiki, "2026-11-27", "a lion roaring"},
	{"commons/Australasian-bittern.wav", "https://upload.wikimedia.org/wikipedia/commons/8/84/Australasian-bittern.wav",
		"https://commons.wikimedia.org/wiki/File:Australasian-bittern.wav", "Department of Conservation (NZ), via Wikimedia Commons", ccBY4, "2026-11-27", "an Australasian bittern booming"},
	{"commons/Elephant_voice_-_trumpeting.ogg", "https://upload.wikimedia.org/wikipedia/commons/4/40/Elephant_voice_-_trumpeting.ogg",
		"https://commons.wikimedia.org/wiki/File:Elephant_voice_-_trumpeting.ogg", "த*உழவன் (Info-farmer), via Wikimedia Commons", cc0, "2026-11-27", "an elephant trumpeting"},
	{"commons/161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", "https://upload.wikimedia.org/wikipedia/commons/5/5a/161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav",
		"https://commons.wikimedia.org/wiki/File:161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", "Felix Blume (Freesound sound 161691), via Wikimedia Commons", cc0, "2026-11-27", "dolphins screaming under water, Caribbean Sea"},
	{"commons/157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav", "https://upload.wikimedia.org/wikipedia/commons/2/25/157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav",
		"https://commons.wikimedia.org/wiki/File:157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav", "Felix Blume (Freesound sound 157763), via Wikimedia Commons", cc0, "2026-11-27", "a donkey braying, south of France"},
	{"commons/439280_schots_angry-tiger.wav", "https://upload.wikimedia.org/wikipedia/commons/2/29/439280_schots_angry-tiger.wav",
		"https://commons.wikimedia.org/wiki/File:439280_schots_angry-tiger.wav", "schots (Freesound sound 439280), via Wikimedia Commons", cc0, "2026-11-27", "a caged tiger growling and snarling"},
	{"commons/Pavo_cristatus_(call).ogg", "https://upload.wikimedia.org/wikipedia/commons/0/05/Pavo_cristatus_%28call%29.ogg",
		"https://commons.wikimedia.org/wiki/File:Pavo_cristatus_(call).ogg", "Ke4roh, via Wikimedia Commons", pdWiki, "2026-11-27", "a peacock calling, Denver Zoo"},
	{"commons/Arctocephalus_forsteri_-_sound.ogg", "https://upload.wikimedia.org/wikipedia/commons/5/56/Arctocephalus_forsteri_-_sound.ogg",
		"https://commons.wikimedia.org/wiki/File:Arctocephalus_forsteri_-_sound.ogg", "Department of Conservation (NZ), via Wikimedia Commons", ccBY4, "2026-11-27", "New Zealand fur seals growling, Stephens Island"},
	{"commons/Geese_Honking_(loud).ogg", "https://upload.wikimedia.org/wikipedia/commons/a/aa/Geese_Honking_%28loud%29.ogg",
		"https://commons.wikimedia.org/wiki/File:Geese_Honking_(loud).ogg", "U.S. Fish and Wildlife Service, via Wikimedia Commons", pdWiki, "2026-11-27", "Canada geese honking"},
	{"commons/Great_Blue_Heron.ogg", "https://upload.wikimedia.org/wikipedia/commons/2/23/Great_Blue_Heron.ogg",
		"https://commons.wikimedia.org/wiki/File:Great_Blue_Heron.ogg", "U.S. National Park Service (Wind Cave National Park), via Wikimedia Commons", pdNPS, "2026-11-27", "a great blue heron croaking"},
}

// clip is a part of a recording cut out to be used: assets/voices/src/<name>.qoa.
type clip struct {
	name   string
	src    string  // the source's file
	t0, t1 float64 // seconds
	clean  float64 // how hard to take out the background (0: not at all)
	hp, lp float64 // band limits, Hz (0: none)
}

var clips = []clip{
	// Bison bellows: the horned herds, the tyrant's chest.
	{"bison_a1", "nps/nri-BisonYELL.mp3", 0.88, 2.4, 0.6, 40, 0},
	{"bison_a2", "nps/nri-BisonYELL.mp3", 2.58, 4.3, 0.6, 40, 0},
	{"bison_a3", "nps/nri-BisonYELL.mp3", 5.28, 6.2, 0.6, 40, 0},
	{"bison_b1", "nps/yell-YELLMM8K2005914Bison.mp3", 0.95, 3.1, 0.6, 40, 0},
	{"bison_b2", "nps/yell-YELLMM8K2005914Bison.mp3", 4.78, 6.5, 0.6, 40, 0},
	{"bison_b3", "nps/yell-YELLMM8K2005914Bison.mp3", 9.2, 10.1, 0.6, 40, 0},
	{"bison_c1", "nps/yell-YELLMM8K2005918Bison.mp3", 5.95, 7.5, 0.6, 40, 0},
	{"bison_c2", "nps/yell-YELLMM8K2005918Bison.mp3", 9.6, 10.9, 0.6, 40, 0},
	{"bison_c3", "nps/yell-YELLMM8K2005918Bison.mp3", 20.35, 22.0, 0.6, 40, 0},
	{"bison_c4", "nps/yell-YELLMM8K2005918Bison.mp3", 22.95, 25.05, 0.6, 40, 0},
	{"bison_c5", "nps/yell-YELLMM8K2005918Bison.mp3", 26.05, 28.25, 0.6, 40, 0},
	// Elk bugles: slowed, a trumpeting scream.
	{"elk_1", "nps/yell-ElkBugle1.mp3", 2.8, 5.9, 0.5, 60, 0},
	{"elk_2", "nps/yell-ElkBugle1.mp3", 6.28, 8.8, 0.5, 60, 0},
	{"elk_3", "nps/yell-ElkBugle2.mp3", 1.9, 5.1, 0.5, 60, 0},
	{"elk_4", "nps/yell-ElkBugle2.mp3", 12.75, 16.3, 0.5, 60, 0},
	{"elk_5", "nps/yell-YELLElkBullandCows20151018T012.mp3", 118.3, 121.6, 0.5, 60, 0},
	{"elk_6", "nps/yell-YELLElkBullandCows20151018T012.mp3", 92.8, 96.6, 0.5, 60, 0},
	{"elk_7", "nps/yell-YELLElkChorus150922T04ms01.mp3", 72.75, 76.3, 0.5, 60, 0},
	{"elk_8", "nps/yell-YELLElkChorus150922T04ms01.mp3", 87.65, 90.4, 0.5, 60, 0},
	{"elk_9", "nps/nri-ElkBuglingGRSA.mp3", 1.45, 3.95, 0.5, 60, 0},
	{"elk_10", "nps/nri-ElkBuglingGRSA.mp3", 7.4, 12.0, 0.5, 60, 0},
	{"elkcalf_1", "nps/yell-YELLMJ23ElkCalf20051116.mp3", 0.15, 2.2, 0.5, 200, 0},
	{"elkcalf_2", "nps/yell-YELLMJ23ElkCalf20051116.mp3", 3.4, 5.75, 0.5, 200, 0},
	{"elkcalf_3", "nps/yell-YELLMJ23ElkCalf20051116.mp3", 6.7, 9.0, 0.5, 200, 0},
	// The grizzly: a roar and a long low moan; cubs bawling.
	{"grizzly_1", "nps/yell-YELLGrizzlyBearVocalizations20160520Android1.mp3", 14.8, 20.2, 0.4, 40, 0},
	{"grizzly_2", "nps/yell-YELLGrizzlyBearVocalizations20160520Android1.mp3", 40.95, 47.3, 0.4, 30, 0},
	{"cub_1", "nps/nri-BearwithCubsDENA.mp3", 0.4, 1.65, 0.5, 120, 0},
	{"cub_2", "nps/nri-BearwithCubsDENA.mp3", 3.0, 4.15, 0.5, 120, 0},
	{"cub_3", "nps/nri-BearwithCubsDENA.mp3", 4.3, 5.05, 0.5, 120, 0},
	// Alligators bellowing: only the low end is clean of the swamp.
	{"gator_1", "nps/nri-AlligatorEVER1.mp3", 0.9, 2.7, 0.9, 25, 900},
	{"gator_2", "nps/nri-AlligatorEVER1.mp3", 7.85, 10.6, 0.9, 25, 900},
	{"gator_3", "nps/nri-AlligatorEVER1.mp3", 15.2, 17.8, 0.9, 25, 900},
	{"gator_4", "nps/nri-AlligatorEVER1.mp3", 22.15, 24.5, 0.9, 25, 900},
	// Birds.
	{"raven_1", "nps/nri-CommonRavenGRTE.mp3", 3.7, 5.55, 0.6, 150, 0},
	{"raven_2", "nps/yell-RavenBlackSandBasin.mp3", 4.4, 6.4, 0.5, 150, 0},
	{"raven_3", "nps/yell-RavenBlackSandBasin.mp3", 6.45, 7.85, 0.5, 150, 0},
	{"raven_4", "nps/yell-RavenBlackSandBasin.mp3", 8.35, 8.85, 0.5, 150, 0},
	{"osprey_1", "nps/nri-OspreyGRTE.mp3", 0.35, 1.8, 0.6, 300, 0},
	{"eagle_1", "nps/nri-BaldEagleYELL.mp3", 0.1, 2.45, 0.7, 400, 0},
	{"peregrine_1", "nps/nri-PeregrineandRaven.mp3", 4.7, 5.55, 0.6, 400, 0},
	{"peregrine_2", "nps/nri-PeregrineandRaven.mp3", 5.6, 6.7, 0.6, 400, 0},
	{"peregrine_3", "nps/nri-PeregrineandRaven.mp3", 6.75, 8.95, 0.6, 400, 0},
	{"crane_1", "nps/yell-YELLFLBCSACR20075171.mp3", 9.0, 11.35, 0.5, 120, 0},
	{"crane_2", "nps/yell-YELLFLBCSACR20075171.mp3", 11.42, 12.95, 0.5, 120, 0},
	{"crane_3", "nps/yell-YELLFLBCSACR20075171.mp3", 7.5, 8.75, 0.5, 120, 0},
	{"owl_1", "nps/nri-SpottedOwlYOSE.mp3", 1.8, 2.7, 0.5, 150, 0},
	{"owl_2", "nps/nri-SpottedOwlYOSE.mp3", 3.78, 4.4, 0.5, 150, 0},
	{"anhinga_1", "nps/nri-AnhingaEVER.mp3", 0.3, 2.1, 0.6, 200, 4500},
	{"anhinga_2", "nps/nri-AnhingaEVER.mp3", 4.0, 5.65, 0.6, 200, 4500},
	{"anhinga_3", "nps/nri-AnhingaEVER.mp3", 7.42, 9.5, 0.6, 200, 4500},
	{"geese_1", "commons/Geese_Honking_(loud).ogg", 0.0, 3.3, 0.5, 150, 0},
	{"geese_2", "commons/Geese_Honking_(loud).ogg", 3.5, 6.4, 0.5, 150, 0},
	{"geese_3", "commons/Geese_Honking_(loud).ogg", 6.45, 8.65, 0.5, 150, 0},
	{"heron_1", "commons/Great_Blue_Heron.ogg", 0.1, 0.5, 0.5, 120, 0},
	{"heron_2", "commons/Great_Blue_Heron.ogg", 0.85, 1.18, 0.5, 120, 0},
	{"heron_3", "commons/Great_Blue_Heron.ogg", 1.58, 1.88, 0.5, 120, 0},
	{"heron_4", "commons/Great_Blue_Heron.ogg", 2.5, 2.8, 0.5, 120, 0},
	// Foxes and frogs.
	{"fox_1", "nps/yell-YELLMJ23200837redfox.mp3", 0.6, 1.4, 0.4, 200, 0},
	{"fox_2", "nps/yell-YELLMJ23200837redfox.mp3", 7.55, 8.2, 0.4, 200, 0},
	{"pigfrog_1", "nps/nri-PigFrogEvergladesNP.mp3", 0.05, 1.85, 0.4, 30, 0},
	// Lions.
	{"lion_1", "commons/Lion_raring-sound1TamilNadu178.ogg", 0.38, 2.35, 0.5, 30, 0},
	{"lion_2", "commons/Lion_raring-sound1TamilNadu178.ogg", 4.85, 6.55, 0.5, 30, 0},
	// A bittern's boom: the emu-like booms of the Thescelosaurus and the Anzu.
	{"bittern_1", "commons/Australasian-bittern.wav", 1.0, 1.95, 0.4, 40, 0},
	{"bittern_2", "commons/Australasian-bittern.wav", 3.02, 3.95, 0.4, 40, 0},
	{"bittern_3", "commons/Australasian-bittern.wav", 7.35, 8.15, 0.4, 40, 0},
	{"bittern_4", "commons/Australasian-bittern.wav", 9.02, 9.97, 0.4, 40, 0},
	// A donkey's bray, far down: the long-necks' call, as in the films.
	{"donkey_hee", "commons/157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav", 6.0, 8.6, 0.3, 60, 0},
	{"donkey_hee2", "commons/157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav", 0.05, 3.0, 0.3, 60, 0},
	{"donkey_haw", "commons/157763_felix-blume_a-donkey-is-braying-in-his-enclosure-in-south-of-france.wav", 3.55, 5.1, 0.3, 60, 0},
	// An elephant.
	{"elephant_1", "commons/Elephant_voice_-_trumpeting.ogg", 0.04, 1.25, 0.4, 60, 0},
	// Dolphins screaming and a peacock: the raptors' and the flyers' screams.
	{"dolphin_1", "commons/161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", 16.25, 17.25, 0.5, 200, 0},
	{"dolphin_2", "commons/161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", 38.3, 39.85, 0.5, 200, 0},
	{"dolphin_3", "commons/161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", 11.85, 13.4, 0.5, 200, 0},
	{"dolphin_4", "commons/161691_felixblume_dolphin-screaming-underwater-in-caribbean-sea-mexico.wav", 27.95, 29.45, 0.5, 200, 0},
	{"peacock_1", "commons/Pavo_cristatus_(call).ogg", 1.05, 2.0, 0.3, 200, 0},
	{"peacock_2", "commons/Pavo_cristatus_(call).ogg", 2.4, 3.45, 0.3, 200, 0},
	{"peacock_3", "commons/Pavo_cristatus_(call).ogg", 3.93, 4.9, 0.3, 200, 0},
	{"peacock_4", "commons/Pavo_cristatus_(call).ogg", 7.3, 8.2, 0.3, 200, 0},
	{"furseal_2", "commons/Arctocephalus_forsteri_-_sound.ogg", 24.3, 26.05, 0.6, 60, 0},
	{"furseal_3", "commons/Arctocephalus_forsteri_-_sound.ogg", 26.7, 27.7, 0.6, 60, 0},
	// A tiger's snarls and growls: the sabre-tooth, the raptors' bite.
	{"tiger_1", "commons/439280_schots_angry-tiger.wav", 16.45, 17.95, 0.4, 60, 0},
	{"tiger_2", "commons/439280_schots_angry-tiger.wav", 34.3, 35.95, 0.4, 60, 0},
	{"tiger_3", "commons/439280_schots_angry-tiger.wav", 52.75, 54.75, 0.4, 60, 0},
	{"tiger_4", "commons/439280_schots_angry-tiger.wav", 60.0, 60.6, 0.4, 60, 0},
	{"tiger_5", "commons/439280_schots_angry-tiger.wav", 11.05, 12.2, 0.4, 60, 0},
	// The sea.
	{"seal_1", "noaa/Phvi-roar-NOAA-PAGroup-01-harbor-seal-clip.wav", 2.4, 3.3, 0.6, 30, 0},
	{"seal_2", "noaa/Phvi-roar-NOAA-PAGroup-01-harbor-seal-clip.wav", 4.8, 5.95, 0.6, 30, 0},
}
