package main

import (
	"bytes"
	"log"
	"net"
)

const (
	defaultBufSize = 65536
	minRequestLen  = 8
)

type Tunnel struct {
	lconn  net.Conn // the connection from internal
	rconn  net.Conn // the connection to external
	closed bool
}

func refuseConn(t *Tunnel) {
	buf := []byte{0, 0x5b, 0, 0, 0, 0, 0, 0}
	t.lconn.Write(buf)
	t.closed = true
}

func successConn(t *Tunnel) {
	buf := []byte{0, 0x5a, 0, 0, 0, 0, 0, 0}
	t.lconn.Write(buf)
}

/*func lookupAddr(host string) (net.IP, error) {
	ip, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return net.IP{}, err
	}

	return ip.IP.To4(), nil
}

func parseAddr(addr string) (host string, iport int, err error) {
	var port string

	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}

	iport, err = strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}

	return
}*/

func servExternalTunnel(t *Tunnel) {
	defer t.rconn.Close()

	buf := make([]byte, defaultBufSize)

	for !t.closed {
		n, err := t.rconn.Read(buf)
		log.Printf("read %v bytes from external server %v", n, t.rconn.RemoteAddr())
		if n == 0 {
			t.closed = true
			return
		}

		if err != nil {
			log.Printf("read from external server %v", err)
			return
		}

		t.lconn.Write(buf[:n])
	}
}

func connectExternal(buf []byte, bufSize int, t *Tunnel, externlf string) bool {
	end := bytes.IndexByte(buf[8:], byte(0))

	if end < -1 {
		log.Printf("cannot find '\\0'")
		return false
	}

	ver := buf[0]
	cmd := buf[1]
	port := int(buf[2])<<8 + int(buf[3])
	ip := net.IPv4(buf[4], buf[5], buf[6], buf[7])

	log.Printf("ver=%v cmd=%v external addr %v:%v\n", ver, cmd, ip, port)

	external := &net.TCPAddr{IP: ip, Port: port}
	internal := &net.TCPAddr{IP: net.ParseIP(externlf)}

	c, err := net.DialTCP("tcp", internal, external)
	if err != nil {
		log.Printf("DialTCP", err)
		refuseConn(t)
		return false
	}

	log.Printf("has connected to external %s", external)

	t.rconn = c
	if end < bufSize {
		t.rconn.Write(buf[end:])
	}

	go servExternalTunnel(t)
	return true
}

func servInternalTunnel(lconn net.Conn, externlf string) {
	t := new(Tunnel)
	t.lconn = lconn
	defer t.lconn.Close()

	buf := make([]byte, defaultBufSize)
	bufSize := 0

	for !t.closed {
		n, _ := t.lconn.Read(buf[bufSize:])
		log.Printf("read %v bytes from internal %v", n, t.lconn.RemoteAddr())
		if n == 0 {
			t.closed = true
			return
		}

		if t.rconn != nil {
			log.Printf("write %v bytes to external server %v", n, t.rconn.RemoteAddr())
			t.rconn.Write(buf[:n+bufSize])
			bufSize = 0
			continue
		}

		if n+bufSize <= minRequestLen {
			log.Printf("need to recv more data")
			continue
		}

		if connectExternal(buf, bufSize, t, externlf) {
			bufSize = 0
			successConn(t)
		}
	}
}

func start(addr string, externlf string) {
	addrTCP, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.ListenTCP("tcp", addrTCP)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("socks server bind %v", addrTCP)

	for {
		c, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("connection from %v", c.RemoteAddr().String())
		go servInternalTunnel(c, externlf)
	}
}
