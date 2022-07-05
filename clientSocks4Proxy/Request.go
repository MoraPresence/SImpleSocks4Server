package main

import (
	"bytes"
	"encoding/binary"
	"net"
)

type request struct {
	Host string
	Port int
	IP   net.IP

	err error
	buf bytes.Buffer
}

func (r *request) write(b []byte) {
	if r.err == nil {
		_, r.err = r.buf.Write(b)
	}
}

func (r *request) writeString(s string) {
	if r.err == nil {
		_, r.err = r.buf.WriteString(s)
	}
}

func (r *request) writeBigEndian(data interface{}) {
	if r.err == nil {
		r.err = binary.Write(&r.buf, binary.BigEndian, data)
	}
}

func (r request) Bytes() ([]byte, error) {
	r.write([]byte{socksVersion, socksConnect}) //Номер версии SOCKS, 1 байт + Код команды: 0x01 = установка TCP/IP соединения 0x02 = (binding)
	r.writeBigEndian(uint16(r.Port))            //Номер порта 2 байта
	r.writeBigEndian(r.IP.To4())                //4 байта	IP-адрес
	r.write([]byte{0})

	return r.buf.Bytes(), r.err
}
