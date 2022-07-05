package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
)

type Config struct {
	ListenIf   string `json:listenIf`
	ListenPort string `json:listenPort`
	ExternIf   string `json:externIf`
	LogFile    string `json:logFile`
}

func GetInternalIP(netInt string) string {
	itf, _ := net.InterfaceByName(netInt) //here your interface
	item, _ := itf.Addrs()
	var ip net.IP
	for _, addr := range item {
		switch v := addr.(type) {
		case *net.IPNet:
			if !v.IP.IsLoopback() {
				if v.IP.To4() != nil { //Verify if IP is IPV4
					ip = v.IP
				}
			}
		}
	}
	if ip != nil {
		return ip.String()
	} else {
		return ""
	}
}

func main() {
	jfile, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}

	data := make([]Config, 1)

	err = json.Unmarshal(jfile, &data)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(data[0].LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	ipInternal := GetInternalIP(data[0].ListenIf)
	ipExternal := GetInternalIP(data[0].ListenIf)

	log.SetOutput(f)
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	start(ipInternal+":"+data[0].ListenPort, ipExternal)
}
