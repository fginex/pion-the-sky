package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"golang.org/x/image/vp8"
)

// WebRTCService server implementation
type WebRTCService struct {
	api    *webrtc.API
	config webrtc.Configuration
	m      webrtc.MediaEngine
	ac     *webrtc.RTPCodec
	vc     *webrtc.RTPCodec
	Videos map[string]*bytes.Buffer //TODO: Make this a sync.Map
}

// CreateNewWebRTCService creates a new webrtc server instance
func CreateNewWebRTCService(videoCodec *webrtc.RTPCodec) (*WebRTCService, error) {

	svc := WebRTCService{Videos: make(map[string]*bytes.Buffer)}
	svc.ac = webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000)
	svc.vc = videoCodec

	// Selected Codecs
	svc.m.RegisterCodec(svc.ac)
	svc.m.RegisterCodec(svc.vc)

	svc.api = webrtc.NewAPI(webrtc.WithMediaEngine(svc.m))

	svc.config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	log.Printf("WebRTC services started with [Audio:%s, Video:%s]\n", svc.ac.Name, svc.vc.Name)
	return &svc, nil
}

// CreateRecordingConnection creates a new webrtc peer connection on the server for recording and streaming playback.
func (svc *WebRTCService) CreateRecordingConnection(client *PeerClient) error {
	var err error

	// Create a new peer connection
	client.pc, err = svc.api.NewPeerConnection(svc.config)
	if err != nil {
		return err
	}

	// Create receive track
	inputTrack, err := client.pc.NewTrack(svc.vc.PayloadType, rand.Uint32(), "video", "pion")
	if err != nil {
		log.Printf("Client pt=%d pion-pt:%d\n", client.pt, svc.vc.PayloadType)
		panic(err)
	}

	// Add this newly created track to the PeerConnection
	if _, err = client.pc.AddTrack(inputTrack); err != nil {
		panic(err)
	}

	// Handler - Process audio/video as it is received
	client.pc.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		log.Printf("Client %s %s track ready\n", client.id, track.Codec().Name)

		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				if client.IsClosed() {
					return
				}

				err := client.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
				if err != nil {
					fmt.Printf("OnTrack ticker exiting for client %s (%s)\n", client.id, err)
					return
				}
			}
		}()

		go client.recordTrack(track)
	})

	// Handler - Detect connects, disconnects & closures
	client.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Client %s connection State has changed %s \n", client.id, connectionState.String())

		if connectionState == webrtc.ICEConnectionStateConnected {
			log.Printf("Client %s connected to webrtc services as peer.\n", client.id)
		} else if connectionState == webrtc.ICEConnectionStateFailed ||
			connectionState == webrtc.ICEConnectionStateDisconnected ||
			connectionState == webrtc.ICEConnectionStateClosed {

			log.Printf("Client %s disconnected from webrtc services as peer.\n", client.id)
		}
	})

	err = client.startServerSession()
	if err != nil {
		return err
	}

	return nil
}

// CreatePlaybackConnection creates a new webrtc peer connection on the server for recording and streaming playback.
func (svc *WebRTCService) CreatePlaybackConnection(client *PeerClient) error {
	var err error

	// Create a new peer connection
	client.pc, err = svc.api.NewPeerConnection(svc.config)
	if err != nil {
		return err
	}

	// Create Track that we send video back to browser on
	outputTrack, err := client.pc.NewTrack(svc.vc.PayloadType, rand.Uint32(), "video", "pion")
	if err != nil {
		panic(err)
	}

	// Add this newly created track to the PeerConnection
	if _, err = client.pc.AddTrack(outputTrack); err != nil {
		panic(err)
	}

	// Handler - Detect connects, disconnects & closures
	client.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Client %s connection State has changed %s \n", client.id, connectionState.String())

		if connectionState == webrtc.ICEConnectionStateConnected {
			log.Printf("Client %s connected to webrtc services as peer.\n", client.id)

			go client.streamVideoToTrack(outputTrack)

		} else if connectionState == webrtc.ICEConnectionStateFailed ||
			connectionState == webrtc.ICEConnectionStateDisconnected ||
			connectionState == webrtc.ICEConnectionStateClosed {

			log.Printf("Client %s disconnected from webrtc services as peer.\n", client.id)
		}
	})

	err = client.startServerSession()
	if err != nil {
		return err
	}

	return nil
}

// SaveVideo stores video packets in a globally available map for streaming playback
func (svc *WebRTCService) SaveVideo(id string, packets *bytes.Buffer) {
	svc.Videos[id] = packets
	log.Printf("Video saved in memory for Client %s. %d bytes.\n", id, packets.Len())
	log.Printf("%d total videos in memory.\n", svc.VideoCount())
}

// VideoCount returns the number or stored videos for streaming playback
func (svc *WebRTCService) VideoCount() int {
	return len(svc.Videos)
}

// RTPToString compiles the rtp header fields into a string for logging.
func RTPToString(pkt *rtp.Packet) string {
	return fmt.Sprintf("RTP:{Version:%d Padding:%v Extension:%v Marker:%v PayloadOffset:%d PayloadType:%d SequenceNumber:%d Timestamp:%d SSRC:%d CSRC:%v ExtensionProfile:%d ExtensionPayload:%s PayloadLen:%d}",
		pkt.Version,
		pkt.Padding,
		pkt.Extension,
		pkt.Marker,
		pkt.PayloadOffset,
		pkt.PayloadType,
		pkt.SequenceNumber,
		pkt.Timestamp,
		pkt.SSRC,
		pkt.CSRC,
		pkt.ExtensionProfile,
		pkt.ExtensionPayload,
		len(pkt.Payload),
	)
}

// VP8FrameHeaderToString compiles a vp8 video frame header fields into a string for logging.
func VP8FrameHeaderToString(fh *vp8.FrameHeader) string {
	return fmt.Sprintf("VP8:{KeyFrame:%v VersionNumber:%d ShowFrame:%v FirstPartitionLen:%d Width:%d Height:%d XScale:%d YScale:%d}",
		fh.KeyFrame,
		fh.VersionNumber,
		fh.ShowFrame,
		fh.FirstPartitionLen,
		fh.Width,
		fh.Height,
		fh.XScale,
		fh.YScale,
	)

}

// SaveAsPNG saves the specified image as a png file.
func SaveAsPNG(img *image.YCbCr, fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	err = png.Encode(f, (*image.YCbCr)(img))
	if err != nil {
		return err
	}

	log.Printf("PNG file saved: %s\n", fn)
	return nil
}

// ModAnswer modifies the remote session description to work around known issues.
func ModAnswer(sd *webrtc.SessionDescription) *webrtc.SessionDescription {

	// https://stackoverflow.com/questions/47990094/failed-to-set-remote-video-description-send-parameters-on-native-ios
	sd.SDP = strings.Replace(sd.SDP, "42001f", "42e01f", -1)

	return sd
}
