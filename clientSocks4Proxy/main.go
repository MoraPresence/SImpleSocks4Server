package main

import (
	"errors"
	"golang.org/x/net/proxy"
	"log"
	"net/url"
)

func main() {
	addr, _ := url.Parse("socks4://192.168.234.1:8081")

	dialer, err := proxy.FromURL(addr, proxy.Direct)
	conn, err := dialer.Dial("tcp", "google.com:443")
	if err != nil {
		// handle error
		if errors.Is(err, errors.New("123")) {
			log.Printf("invalid proxy server %s", addr)
			return
		}
		if errors.Is(err, errors.New("456")) {
			log.Printf("google.com:80: %s", err)
			return
		}
	}

	_ = conn
}
