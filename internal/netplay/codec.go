package netplay

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/gob"
	"io"
	"sync/atomic"

	"survivalgame/internal/protocol"
)

// counter counts bytes that crossed the wire; a nil counter counts nothing.
type counter struct{ n atomic.Int64 }

func (c *counter) add(n int) {
	if c != nil {
		c.n.Add(int64(n))
	}
}

func (c *counter) get() int64 {
	if c == nil {
		return 0
	}
	return c.n.Load()
}

// downEncoder is the server's end of the downstream: ServerMsg in, one
// compressed gob stream out. Not safe for concurrent use.
type downEncoder struct {
	out *bufio.Writer
	fl  *flate.Writer
	enc *gob.Encoder
}

// compressLevel: measured (TestBandwidth), the fastest level costs four times
// the bytes on snapshots, which look so much like the last one that only a
// level that searches the whole window finds the match; the tile map is the
// only big message
// and is sent once per join.
const compressLevel = flate.DefaultCompression

func newDownEncoder(w io.Writer) *downEncoder {
	out := bufio.NewWriterSize(w, writeChunk)
	fl, _ := flate.NewWriter(out, compressLevel) // the level is valid
	return &downEncoder{out: out, fl: fl, enc: gob.NewEncoder(fl)}
}

// send encodes m and pushes it all the way out, so the other side can decode
// it without waiting for more.
func (d *downEncoder) send(m protocol.ServerMsg) error {
	if err := d.enc.Encode(m); err != nil {
		return err
	}
	if err := d.fl.Flush(); err != nil {
		return err
	}
	return d.out.Flush()
}

// downDecoder is the client's end of the downstream.
type downDecoder struct {
	dec *gob.Decoder
}

func newDownDecoder(r io.Reader) *downDecoder {
	fr := flate.NewReader(bufio.NewReaderSize(r, writeChunk))
	return &downDecoder{dec: gob.NewDecoder(fr)}
}

// next decodes one message. Every message goes into a fresh value, because
// gob leaves the fields it did not send untouched.
func (d *downDecoder) next() (protocol.ServerMsg, error) {
	var m protocol.ServerMsg
	err := d.dec.Decode(&m)
	return m, err
}

// encodeClient makes one upstream frame body: a self-contained gob, so the
// server decodes each frame on its own.
func encodeClient(m protocol.ClientMsg) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeClient(b []byte) (protocol.ClientMsg, error) {
	var m protocol.ClientMsg
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&m)
	return m, err
}
