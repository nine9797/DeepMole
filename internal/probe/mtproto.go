package probe

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"time"
)

const KindMTProto = "mtproto"

const (
	reqPQMulti = 0xbe7e8ef1
	resPQ      = 0x05162463
)

func checkMTProto(ctx context.Context, addr string, timeout time.Duration) (int, error) {
	addr = strings.TrimPrefix(addr, KindMTProto+"://")
	d := &net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return 0, err
	}
	body := make([]byte, 20)
	binary.LittleEndian.PutUint32(body, reqPQMulti)
	copy(body[4:], nonce[:])

	msg := make([]byte, 20+len(body))
	binary.LittleEndian.PutUint64(msg[8:], uint64(time.Now().Unix())<<32)
	binary.LittleEndian.PutUint32(msg[16:], uint32(len(body)))
	copy(msg[20:], body)

	frame := make([]byte, 8+len(msg))
	binary.LittleEndian.PutUint32(frame, 0xeeeeeeee)
	binary.LittleEndian.PutUint32(frame[4:], uint32(len(msg)))
	copy(frame[8:], msg)
	if _, err := conn.Write(frame); err != nil {
		return 0, err
	}

	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return 0, err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n < 40 || n > 4096 {
		return 4, errors.New("bad mtproto reply")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return 4, err
	}
	if binary.LittleEndian.Uint32(buf[20:]) != resPQ || string(buf[24:40]) != string(nonce[:]) {
		return 4 + int(n), errors.New("unexpected mtproto reply")
	}
	return 4 + int(n), nil
}
