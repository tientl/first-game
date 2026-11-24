package main

import (
	"bytes"
	"encoding/binary"
	"image"
)

// encodeSyso packs the icons into a Windows resource object (COFF, amd64)
// holding one icon group, for the Go linker to put into the executable:
// saved as cmd/game/rsrc_windows_amd64.syso, it gives the game its icon in
// Explorer. Each image is stored as in the .ico (iconBlob).
//
// The object has one section, .rsrc: the resource directory tree (type,
// then id, then language), a data entry per resource, and the data; each data
// entry's offset is relocated to an address relative to the image.
func encodeSyso(imgs []image.Image) []byte {
	const (
		rtIcon      = 3
		rtGroupIcon = 14
		langEnUS    = 0x0409
		subdir      = 0x80000000
	)
	n := len(imgs)
	blobs := make([][]byte, n+1)
	for i, img := range imgs {
		blobs[i] = iconBlob(img)
	}
	var grp bytes.Buffer // the group: a GRPICONDIR naming the icons by id
	le := binary.LittleEndian
	binary.Write(&grp, le, [3]uint16{0, 1, uint16(n)})
	for i, img := range imgs {
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		grp.Write([]byte{byte(w), byte(h), 0, 0}) // 256 is written as 0
		binary.Write(&grp, le, [2]uint16{1, 32})
		binary.Write(&grp, le, uint32(len(blobs[i])))
		binary.Write(&grp, le, uint16(i+1))
	}
	blobs[n] = grp.Bytes()

	// Where everything goes in the section.
	const dirSize, entrySize, dataEntrySize = 16, 8, 16
	iconType := dirSize + 2*entrySize
	grpType := iconType + dirSize + n*entrySize
	iconLang := grpType + dirSize + entrySize
	grpLang := iconLang + n*(dirSize+entrySize)
	dataEntries := grpLang + dirSize + entrySize
	at := dataEntries + (n+1)*dataEntrySize
	offsets := make([]int, n+1)
	for i, b := range blobs {
		at = (at + 7) &^ 7
		offsets[i] = at
		at += len(b)
	}
	sec := make([]byte, at)

	dir := func(off, ids int) { le.PutUint16(sec[off+14:], uint16(ids)) }
	entry := func(off, id int, to uint32) {
		le.PutUint32(sec[off:], uint32(id))
		le.PutUint32(sec[off+4:], to)
	}
	dir(0, 2)
	entry(dirSize, rtIcon, subdir|uint32(iconType))
	entry(dirSize+entrySize, rtGroupIcon, subdir|uint32(grpType))
	dir(iconType, n)
	for i := 0; i < n; i++ {
		entry(iconType+dirSize+i*entrySize, i+1, subdir|uint32(iconLang+i*(dirSize+entrySize)))
	}
	dir(grpType, 1)
	entry(grpType+dirSize, 1, subdir|uint32(grpLang))
	langs := make([]int, n+1)
	for i := 0; i < n; i++ {
		langs[i] = iconLang + i*(dirSize+entrySize)
	}
	langs[n] = grpLang
	for i, l := range langs {
		dir(l, 1)
		entry(l+dirSize, langEnUS, uint32(dataEntries+i*dataEntrySize))
		d := dataEntries + i*dataEntrySize
		le.PutUint32(sec[d:], uint32(offsets[i])) // relocated below
		le.PutUint32(sec[d+4:], uint32(len(blobs[i])))
		copy(sec[offsets[i]:], blobs[i])
	}

	// The object: file header, section header, the section, its relocations,
	// the symbol table (the section's own symbol) and an empty string table.
	const fileHeader, sectionHeader, relocSize = 20, 40, 10
	raw := fileHeader + sectionHeader
	relocs := raw + len(sec)
	symbols := relocs + (n+1)*relocSize
	var out bytes.Buffer
	binary.Write(&out, le, struct {
		Machine              uint16
		Sections             uint16
		Time, Symbols, Count uint32
		OptHeader, Flags     uint16
	}{Machine: 0x8664, Sections: 1, Symbols: uint32(symbols), Count: 1})
	binary.Write(&out, le, struct {
		Name                          [8]byte
		VirtualSize, VirtualAddress   uint32
		RawSize, RawAt, RelocAt, Line uint32
		Relocs, Lines                 uint16
		Flags                         uint32
	}{
		Name:    [8]byte{'.', 'r', 's', 'r', 'c'},
		RawSize: uint32(len(sec)), RawAt: uint32(raw), RelocAt: uint32(relocs),
		Relocs: uint16(n + 1),
		Flags:  0x40000040, // initialised data, readable
	})
	out.Write(sec)
	for i := range langs {
		binary.Write(&out, le, struct {
			At     uint32
			Symbol uint32
			Type   uint16
		}{uint32(dataEntries + i*dataEntrySize), 0, 3}) // IMAGE_REL_AMD64_ADDR32NB
	}
	binary.Write(&out, le, struct {
		Name          [8]byte
		Value         uint32
		Section       int16
		Type          uint16
		Class, NumAux uint8
	}{Name: [8]byte{'.', 'r', 's', 'r', 'c'}, Section: 1, Class: 3}) // static
	binary.Write(&out, le, uint32(4)) // the string table: just its own size
	return out.Bytes()
}
