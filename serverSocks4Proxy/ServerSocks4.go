package main

import (
	"bytes"
	"log"
	"net"
)

/*

Есть ряд недочетов в плане обработки установки соединения/ошибок. Например, если мы пустим плохой трафик (клиент не по сокс начнёт работать), то сервер это никак не обнаружит и просто подвиснет.

По интенсивности - 10к соединений он не примет, если не поднимать лимиты в системе.

С transparent-прокси, первый шаг - да, через iptables добавляются redirect правила, но обработка "перехваченных соединений" там немного по другому происходит, поэтому и не работает, ssl тут не при чём, т.к. на прикладной уровень мы не поднимаемся)
—————
К общий итог - тестовое принято. Мы пока смотрим ещё кандидатов, поэтому в течении ближайших недель двух дадим обратную связь)
*за исправление с интерфейсами ошибки - + в карму)
*/
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
	_, err := t.lconn.Write(buf)
	if err != nil {
		log.Printf("cannot write in refuseConn: %s", err)
		return
	}
	t.closed = true
}

func successConn(t *Tunnel) {
	buf := []byte{0, 0x5a, 0, 0, 0, 0, 0, 0}
	_, err := t.lconn.Write(buf)
	if err != nil {
		log.Printf("cannot write in successConn: %s", err)
		return
	}
}

func servExternalTunnel(t *Tunnel) {
	defer func(rconn net.Conn) {
		err := rconn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(t.rconn)

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

		_, err = t.lconn.Write(buf[:n])
		if err != nil {
			//log.Printf("cannot write in servExternalTunnel: %s", err)
			return
		}
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
		_, err := t.rconn.Write(buf[end:])
		if err != nil {
			log.Fatal(err)
		}
	}

	go servExternalTunnel(t)
	return true
}

func servInternalTunnel(lconn net.Conn, externlf string) {
	t := new(Tunnel)
	t.lconn = lconn
	defer func(lconn net.Conn) {
		err := lconn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(t.lconn)

	buf := make([]byte, defaultBufSize)
	bufSize := 0

	for !t.closed {
		n, _ := t.lconn.Read(buf[bufSize:])
		log.Printf("read %v bytes from internal %v", n, t.lconn.RemoteAddr())
		if n == 0 {
			t.closed = true
			return
		} else if buf[0] != 0x04 && t.rconn == nil {
			log.Printf("wrong package arrived, not socks4")
			t.closed = true
			return
		}

		if t.rconn != nil {
			log.Printf("write %v bytes to external server %v", n, t.rconn.RemoteAddr())
			_, err := t.rconn.Write(buf[:n+bufSize])
			if err != nil {
				log.Fatal(err)
			}
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
