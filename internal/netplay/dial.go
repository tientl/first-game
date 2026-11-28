package netplay

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"survivalgame/internal/protocol"
)

// DialTimeout is how long Dial waits to reach the server and shake hands.
const DialTimeout = 6 * time.Second

// Conn is the client end of a connection to a room. It satisfies
// client.Conn: the game cannot tell it from server.LocalConn.
type Conn struct {
	nc   net.Conn
	out  chan []byte // upstream frames waiting to be written
	msgs chan protocol.ServerMsg
	done chan struct{}
	once sync.Once

	in, sent counter
}

// Dial connects to a room at addr (see JoinAddr for what it accepts) and
// completes the handshake, so a protocol mismatch comes back as a
// *VersionError before anything else is exchanged. It gives up when ctx is
// cancelled or after DialTimeout.
func Dial(ctx context.Context, addr string) (*Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, DialTimeout)
	defer cancel()
	var d net.Dialer
	nc, err := d.DialContext(ctx, "tcp", JoinAddr(strings.TrimSpace(addr)))
	if err != nil {
		return nil, err
	}
	tuneTCP(nc)
	c := &Conn{
		nc:   nc,
		out:  make(chan []byte, 256),
		msgs: make(chan protocol.ServerMsg, 64),
		done: make(chan struct{}),
	}
	// Cancelling ctx while shaking hands unblocks the reads below.
	stop := context.AfterFunc(ctx, func() { _ = nc.SetDeadline(time.Unix(1, 0)) })
	status, version, msg, err := c.handshake()
	if !stop() { // ctx was cancelled: the deadline is already set
		if err == nil {
			err = ctx.Err()
		}
	}
	if err != nil {
		_ = nc.Close()
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			err = errBadHandshake
		}
		return nil, err
	}
	switch status {
	case statusOK:
	case statusVersion:
		_ = nc.Close()
		return nil, &VersionError{Server: version, Client: protocol.Version, Msg: msg}
	default:
		_ = nc.Close()
		return nil, &RefusedError{Msg: msg}
	}
	_ = nc.SetDeadline(time.Time{})
	go c.readLoop()
	go c.writeLoop()
	return c, nil
}

func (c *Conn) handshake() (status byte, version int, msg string, err error) {
	hello := binary.BigEndian.AppendUint16([]byte(magic), protocol.Version)
	if _, err = (deadlineWriter{c: c.nc, n: &c.sent}).Write(hello); err != nil {
		return
	}
	return readReply(deadlineReader{c: c.nc, idle: DialTimeout, n: &c.in})
}

// Send queues m for the server without blocking; an error means it was not
// sent (the connection is gone, or the link is so slow the queue is full).
func (c *Conn) Send(m protocol.ClientMsg) error {
	select {
	case <-c.done:
		return net.ErrClosed
	default:
	}
	b, err := encodeClient(m)
	if err != nil {
		return err
	}
	if len(b) > maxFrame {
		return fmt.Errorf("netplay: message of %d bytes", len(b))
	}
	select {
	case c.out <- frame(b):
		return nil
	default:
		return errors.New("netplay: send queue full")
	}
}

// Messages delivers what the server sends. The channel is closed when the
// connection is lost or closed.
func (c *Conn) Messages() <-chan protocol.ServerMsg { return c.msgs }

// Close hangs up. It is safe to call more than once.
func (c *Conn) Close() error {
	c.shutdown()
	return nil
}

func (c *Conn) shutdown() {
	c.once.Do(func() {
		close(c.done)
		_ = c.nc.Close()
	})
}

// Traffic is how many bytes this connection has received and sent.
func (c *Conn) Traffic() (in, out int64) { return c.in.get(), c.sent.get() }

func (c *Conn) readLoop() {
	defer close(c.msgs)
	defer c.shutdown()
	dec := newDownDecoder(deadlineReader{c: c.nc, idle: idleTimeout, n: &c.in})
	for {
		m, err := dec.next()
		if err != nil {
			return
		}
		select {
		case c.msgs <- m:
		case <-c.done:
			return
		}
	}
}

func (c *Conn) writeLoop() {
	w := deadlineWriter{c: c.nc, n: &c.sent}
	bw := bufio.NewWriter(w)
	keepalive := time.NewTicker(keepaliveEvery)
	defer keepalive.Stop()
	idle := true
	for {
		select {
		case <-c.done:
			return
		case f := <-c.out:
			// Coalesce what is queued into one write.
			bw.Write(f)
		more:
			for {
				select {
				case f := <-c.out:
					bw.Write(f)
				default:
					break more
				}
			}
			idle = false
			if bw.Flush() != nil {
				c.shutdown()
				return
			}
		case <-keepalive.C:
			if idle {
				bw.Write([]byte{0, 0, 0, 0})
				if bw.Flush() != nil {
					c.shutdown()
					return
				}
			}
			idle = true
		}
	}
}

// Describe turns an error from Dial into a sentence for the player.
func Describe(err error) string {
	var ve *VersionError
	var re *RefusedError
	var ne net.Error
	var de *net.DNSError
	switch {
	case errors.As(err, &ve), errors.As(err, &re):
		return err.Error()
	case errors.Is(err, errBadHandshake):
		return "Địa chỉ này có máy trả lời nhưng không phải phòng Dino Isles (sai cổng?)."
	case errors.Is(err, context.Canceled):
		return "Đã huỷ."
	case errors.As(err, &de):
		return "Không tìm thấy máy có tên hoặc địa chỉ này."
	case errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "refused"):
		return "Máy đó có mở nhưng không có phòng nào ở cổng này: chủ phòng đã bấm Tạo phòng chưa, và đúng cổng chưa?"
	case errors.As(err, &ne) && ne.Timeout(), errors.Is(err, context.DeadlineExceeded):
		return "Không thấy phòng ở địa chỉ này (quá thời gian): kiểm tra IP, tường lửa của chủ phòng, và hai máy có cùng mạng (hoặc cùng Tailscale / ZeroTier) không."
	}
	return "Không vào được phòng: " + err.Error()
}

func versionMessage(server, client int) string {
	return fmt.Sprintf("Khác phiên bản game: máy chủ %d, máy bạn %d. Hai bên cần cùng một bản game.", server, client)
}
