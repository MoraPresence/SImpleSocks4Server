package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	ListenIf   string `json:listenIf`
	ListenPort string `json:listenPort`
	ExternIf   string `json:externIf`
	LogFile    string `json:logFile`
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

	log.SetOutput(f)
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	start(data[0].ListenIf+":"+data[0].ListenPort, data[0].ExternIf)
}
