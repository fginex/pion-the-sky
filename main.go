package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var err error

	log.Println("Media Server starting up.")
	services, err := CreateNewWebRTCService()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("WebRTC services enabled.")

	// TODO: Make the signal server address an arg.
	_, err = CreateNewSignalServer(":8082", services)
	if err != nil {
		log.Fatal(err)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT)
	<-sig
}
