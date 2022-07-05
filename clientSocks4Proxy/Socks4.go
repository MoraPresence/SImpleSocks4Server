package main

import (
	"errors"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/url"
	"strconv"
)

const (
	socksVersion = 0x04
	socksConnect = 0x01

	accessGranted  = 0x5a
	accessRejected = 0x5b

	minRequestLen = 8
)

type socks4 struct {
	url    *url.URL
	dialer proxy.Dialer
}

func (s socks4) parseAddr(addr string) (host string, iport int, err error) {
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
}

func lookupAddr(host string) (net.IP, error) {
	ip, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return net.IP{}, err
	}

	return ip.IP.To4(), nil
}

func init() {
	proxy.RegisterDialerType("socks4", func(u *url.URL, d proxy.Dialer) (proxy.Dialer, error) {
		return socks4{url: u, dialer: d}, nil
	})
}

func (s socks4) Dial(network, addr string) (c net.Conn, err error) {
	if network != "tcp" && network != "tcp4" {
		return nil, errors.New("network should be tcp or tcp4")
	}

	c, err = s.dialer.Dial(network, s.url.Host)
	if err != nil {
		return nil, errors.New("connection to socks4 failed")
	}
	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	host, port, err := s.parseAddr(addr)
	if err != nil {
		return nil, errors.New("wrong address")
	}

	ip := net.IPv4(0, 0, 0, 1)

	if ip, err = lookupAddr(host); err != nil {
		return nil, errors.New("find IP address failed")
	}

	req, err := request{Host: host, Port: port, IP: ip}.Bytes()
	if err != nil {
		return nil, errors.New("write to buffer failed")
	}

	var i int
	i, err = c.Write(req)
	if err != nil {
		return c, errors.New("io error")
	} else if i < minRequestLen {
		return c, errors.New("io error")
	}

	var resp [8]byte
	i, err = c.Read(resp[:])
	if err != nil && err != io.EOF {
		return c, errors.New("io error")
	} else if i != 8 {
		return c, errors.New("io error")
	}

	switch resp[1] {
	case accessGranted:
		return c, nil
	case accessRejected:
		return c, errors.New("request rejected/erroneous")
	default:
		return c, errors.New("invalid data from proxy server")
	}
}
