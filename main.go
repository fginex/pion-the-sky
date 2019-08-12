package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pion/webrtc/v2"
)

func main() {
	var err error

	port := flag.Int("port", 8082, "Endpoint port for the signal server")
	vcodec := flag.String("vcodec", "H264", "Video Codec (H264, VP8, VP9)")
	flag.Parse()

	log.Println("Media Server starting up.")

	var videoCodec *webrtc.RTPCodec

	switch strings.ToUpper(*vcodec) {
	case "H264":
		videoCodec = webrtc.NewRTPH264Codec(webrtc.DefaultPayloadTypeH264, 90000)
	case "VP8":
		videoCodec = webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000)

	//TODO: Issues when running with vp8 (Payload type 98. Getting error: codec payloader not set)
	//case "VP9":
	//	videoCodec = webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000)

	default:
		log.Fatal(fmt.Errorf("unsupported or unrecognized video codec: %s", *vcodec))
		return
	}

	services, err := CreateNewWebRTCService(videoCodec)
	if err != nil {
		log.Fatal(err)
	}

	_, err = CreateNewSignalServer(fmt.Sprintf(":%d", *port), services)
	if err != nil {
		log.Fatal(err)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT)
	<-sig
}
