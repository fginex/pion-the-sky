package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// SignalServer - basic implementation of a webrtc signal server used to locate and connect up with peers.
// In this simple case the peer is the server.
type SignalServer struct {
	services *WebRTCService
}

// CreateNewSignalServer creates a new signal server.
func CreateNewSignalServer(address string, services *WebRTCService) (*SignalServer, error) {

	srv := SignalServer{services: services}

	http.HandleFunc("/", srv.rootHandler)
	http.HandleFunc("/record", srv.recordHandler)
	http.HandleFunc("/play", srv.playHandler)

	http.HandleFunc("/ws", srv.wsHandler)

	go func() {
		log.Printf("Signal server started and listening on %s\n", address)
		err := http.ListenAndServe(address, nil)
		if err != nil {
			panic(err)
		}
	}()

	return &srv, nil
}

func (s *SignalServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Cache the file.
	content, err := ioutil.ReadFile("index.html")
	if err != nil {
		fmt.Println("Could not open file.", err)
	}
	fmt.Fprintf(w, "%s", content)
}

func (s *SignalServer) recordHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Cache the file.
	content, err := ioutil.ReadFile("record.html")
	if err != nil {
		fmt.Println("Could not open file.", err)
	}
	fmt.Fprintf(w, "%s", content)
}

func (s *SignalServer) playHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Cache the file.
	content, err := ioutil.ReadFile("play.html")
	if err != nil {
		fmt.Println("Could not open file.", err)
	}
	fmt.Fprintf(w, "%s", content)
}

func (s *SignalServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Origin not allowed", 403)
		return
	}
	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
	}

	// TODO: keep a map of clients so connections can be managed properly.
	_, err = CreateNewPeerClient(conn, s.services)
	if err != nil {
		log.Printf("wsHandler error %s\n", err)
	}
}
