package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// nativeProbes owns only disposable loopback listeners. The parent first
// verifies they work, so a helper timeout cannot stand for an absent listener.
type nativeProbes struct {
	tcp     net.Listener
	udp     net.PacketConn
	udpDone chan struct{}
}

func openNativeProbes(ctx context.Context) (*nativeProbes, error) {
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	p := &nativeProbes{tcp: tcp}
	p.udp, err = net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		p.close()
		return nil, err
	}
	p.udpDone = make(chan struct{})
	go func() {
		defer close(p.udpDone)
		var packet [1]byte
		for {
			n, from, readErr := p.udp.ReadFrom(packet[:])
			if readErr != nil {
				return
			}
			if n == 1 && packet[0] == 0 {
				_ = p.udp.SetWriteDeadline(time.Now().Add(time.Second))
				_, _ = p.udp.WriteTo(packet[:], from)
			}
		}
	}()
	for i, address := range []string{p.tcp.Addr().String(), p.udp.LocalAddr().String()} {
		network := "tcp4"
		if i == 1 {
			network = "udp4"
		}
		dialer := net.Dialer{Timeout: time.Second}
		connection, checkErr := dialer.DialContext(ctx, network, address)
		if connection != nil {
			if i == 1 {
				_ = connection.SetDeadline(time.Now().Add(time.Second))
				_, checkErr = connection.Write([]byte{0})
				if checkErr == nil {
					var reply [1]byte
					_, checkErr = io.ReadFull(connection, reply[:])
					if checkErr == nil && reply[0] != 0 {
						checkErr = fmt.Errorf("unexpected native UDP probe reply")
					}
				}
			}
			_ = connection.Close()
		}
		if checkErr != nil {
			p.close()
			return nil, fmt.Errorf("verify owned %s listener: %w", network, checkErr)
		}
	}
	return p, nil
}

func (p *nativeProbes) close() {
	_ = p.tcp.Close()
	if p.udp != nil {
		_ = p.udp.Close()
	}
	if p.udpDone != nil {
		<-p.udpDone
	}
}
