package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2/pkg/media/oggreader"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"golang.org/x/image/vp8"
)

var (
	oggPageDuration = time.Millisecond * 20
)

type audioVideoRecording struct {
	audio *bytes.Buffer
	video *bytes.Buffer
}

// WebRTCService server implementation
type WebRTCService struct {
	api         *webrtc.API
	config      webrtc.Configuration
	mediaEngine *webrtc.MediaEngine
	ac          webrtc.RTPCodecParameters
	vc          webrtc.RTPCodecParameters

	logger hclog.Logger

	recordingsLock sync.RWMutex
	recordings     map[string]*audioVideoRecording
}

// CreateNewWebRTCService creates a new webrtc server instance
func CreateNewWebRTCService(logger hclog.Logger) (*WebRTCService, error) {

	svc := WebRTCService{
		mediaEngine:    &webrtc.MediaEngine{},
		recordingsLock: sync.RWMutex{},
		recordings:     map[string]*audioVideoRecording{},
		logger:         logger,
	}
	svc.ac = webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}
	svc.vc = webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}

	// Create a InterceptorRegistry. This is the user configurable RTP/RTCP Pipeline.
	// This provides NACKs, RTCP Reports and other features. If you use `webrtc.NewPeerConnection`
	// this is enabled by default. If you are manually managing You MUST create a InterceptorRegistry
	// for each PeerConnection.
	i := &interceptor.Registry{}

	// Use the default set of Interceptors
	if err := webrtc.RegisterDefaultInterceptors(svc.mediaEngine, i); err != nil {
		fmt.Println("Failed registering an interceptor registry: ", err)
		return nil, err
	}

	// Selected Codecs
	if err := svc.mediaEngine.RegisterCodec(svc.ac, webrtc.RTPCodecTypeAudio); err != nil {
		log.Println("Audio codec setup error: ", err)
		return nil, err
	}
	if err := svc.mediaEngine.RegisterCodec(svc.vc, webrtc.RTPCodecTypeVideo); err != nil {
		log.Println("Video codec setup error: ", err)
		return nil, err
	}

	svc.api = webrtc.NewAPI(webrtc.WithMediaEngine(svc.mediaEngine), webrtc.WithInterceptorRegistry(i))

	svc.config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	log.Printf("WebRTC services started with [Audio:%s, Video:%s]\n", svc.ac.MimeType, svc.vc.MimeType)
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

	// Allow us to receive 1 audio track, and 1 video track
	if _, err = client.pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
		log.Println("Failed adding transciever from audio kind: ", err)
		return err
	} else if _, err = client.pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		log.Println("Failed adding transciever from video kind: ", err)
		return err
	}

	// Handler - Process audio/video as it is received
	client.pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Client %s %s track ready\n", client.id, track.Codec().MimeType)
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				if client.IsClosed() {
					return
				}

				err := client.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if err != nil {
					fmt.Printf("OnTrack ticker exiting for client %s (%s)\n", client.id, err)
					return
				}
			}
		}()
		go func() {
			codec := track.Codec()
			if strings.EqualFold(codec.MimeType, webrtc.MimeTypeOpus) {
				log.Println("Got Opus track, saving to disk as output.opus (48 kHz, 2 channels)")
				client.recordAudioTrack(track)
			} else if strings.EqualFold(codec.MimeType, webrtc.MimeTypeVP8) {
				log.Println("Got VP8 track, saving to disk as output.ivf")
				client.recordVideoTrack(track)
			}
		}()
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
	client.pc, err = svc.api.NewPeerConnection(svc.config)
	if err != nil {
		return err
	}

	svc.recordingsLock.Lock()
	defer svc.recordingsLock.Unlock()

	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	for id, avr := range svc.recordings {
		hasVideo := avr.video != nil && avr.video.Len() > 0
		hasAudio := avr.audio != nil && avr.video.Len() > 0

		if hasVideo {

			log.Println("I have an AV recording: ", id, "video length: ", avr.video.Len())

			// Create a video track
			videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
			if videoTrackErr != nil {
				panic(videoTrackErr)
			}
			rtpSender, videoTrackErr := client.pc.AddTrack(videoTrack)
			if videoTrackErr != nil {
				panic(videoTrackErr)
			}

			go func() {
				rtcpBuf := make([]byte, 1500)
				for {
					if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
						return
					}
				}
			}()

			go func() {
				ivf, header, ivfErr := ivfreader.NewWith(bytes.NewReader(avr.video.Bytes()))
				if ivfErr != nil {
					panic(ivfErr)
				}

				// Wait for connection established
				<-iceConnectedCtx.Done()

				// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
				// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
				//
				// It is important to use a time.Ticker instead of time.Sleep because
				// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
				// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
				duration := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*2300)
				for {
					ctx, ctxCancelFunc := context.WithTimeout(context.Background(), duration)
					frame, header, ivfErr := ivf.ParseNextFrame()
					if errors.Is(ivfErr, io.EOF) {
						fmt.Printf("All video frames parsed and sent")
						ctxCancelFunc()
						break
					}

					if ivfErr != nil {
						panic(ivfErr)
					}

					if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, PacketTimestamp: uint32(header.Timestamp), Duration: time.Second}); ivfErr != nil {
						panic(ivfErr)
					}

					<-ctx.Done()

				}
			}()
		}

		if hasAudio {

			log.Println("I have an AV recording: ", id, "audio length: ", avr.audio.Len())

			// Create a audio track
			audioTrack, audioTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
			if audioTrackErr != nil {
				panic(audioTrackErr)
			}

			rtpSender, audioTrackErr := client.pc.AddTrack(audioTrack)
			if audioTrackErr != nil {
				panic(audioTrackErr)
			}

			// Read incoming RTCP packets
			// Before these packets are returned they are processed by interceptors. For things
			// like NACK this needs to be called.
			go func() {
				rtcpBuf := make([]byte, 1500)
				for {
					if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
						return
					}
				}
			}()

			go func() {
				// Open on oggfile in non-checksum mode.
				ogg, _, oggErr := oggreader.NewWith(bytes.NewReader(avr.audio.Bytes()))
				if oggErr != nil {
					panic(oggErr)
				}

				// Wait for connection established
				<-iceConnectedCtx.Done()

				// Keep track of last granule, the difference is the amount of samples in the buffer
				var lastGranule uint64

				// It is important to use a time.Ticker instead of time.Sleep because
				// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
				// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
				ticker := time.NewTicker(oggPageDuration)
				for ; true; <-ticker.C {
					pageData, pageHeader, oggErr := ogg.ParseNextPage()
					if errors.Is(oggErr, io.EOF) {
						fmt.Printf("All audio pages parsed and sent")
						ticker.Stop()
						break
					}

					if oggErr != nil {
						svc.logger.Error("Ogg error", "reason", oggErr.Error())
						continue
					}

					// The amount of samples is the difference between the last and current timestamp
					sampleCount := float64(pageHeader.GranulePosition - lastGranule)
					lastGranule = pageHeader.GranulePosition
					sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond

					if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
						panic(oggErr)
					}
				}
			}()
		}

		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected

		client.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			fmt.Printf("Connection State has changed %s \n", connectionState.String())
			if connectionState == webrtc.ICEConnectionStateConnected {
				iceConnectedCtxCancel()
			}
		})

		// Set the handler for Peer connection state
		// This will notify you when the peer has connected/disconnected
		client.pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
			fmt.Printf("Peer Connection State has changed: %s\n", s.String())

			if s == webrtc.PeerConnectionStateFailed {
				// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
				// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
				// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
				fmt.Println("Peer Connection has gone to failed exiting")
				//os.Exit(0)
			}
		})

		// TODO: screw that ... play first one only for now....
		break

	}

	err = client.startServerSession()
	if err != nil {
		return err
	}

	return nil // TODO: fix context leak...
}

func (svc *WebRTCService) ensureAudioVideoRecording(id string) int {
	svc.recordingsLock.Lock()
	defer svc.recordingsLock.Unlock()
	if _, ok := svc.recordings[id]; !ok {
		svc.recordings[id] = &audioVideoRecording{}
	}
	return len(svc.recordings)
}

// SaveAudio stores audio packets in a globally available map for streaming playback
func (svc *WebRTCService) SaveAudio(id string, packets *bytes.Buffer) {
	nrecordings := svc.ensureAudioVideoRecording(id)
	svc.recordings[id].audio = packets
	log.Printf("Audio saved in memory for Client %s. %d bytes.\n", id, packets.Len())
	log.Printf("%d total recordings in memory.\n", nrecordings)
}

// SaveVideo stores video packets in a globally available map for streaming playback
func (svc *WebRTCService) SaveVideo(id string, packets *bytes.Buffer) {
	nrecordings := svc.ensureAudioVideoRecording(id)
	svc.recordings[id].video = packets
	log.Printf("Video saved in memory for Client %s. %d bytes.\n", id, packets.Len())
	log.Printf("%d total recordings in memory.\n", nrecordings)
}

// HasRecordings returns true if there is at least one recording available for playback.
func (svc *WebRTCService) HasRecordings() bool {
	svc.recordingsLock.Lock()
	defer svc.recordingsLock.Unlock()
	return len(svc.recordings) > 0
}

// RTPToString compiles the rtp header fields into a string for logging.
func RTPToString(pkt *rtp.Packet) string {
	return fmt.Sprintf("RTP:{Version:%d Padding:%v Extension:%v Marker:%v PayloadType:%d SequenceNumber:%d Timestamp:%d SSRC:%d CSRC:%v ExtensionProfile:%d PayloadLen:%d}",
		pkt.Version,
		pkt.Padding,
		pkt.Extension,
		pkt.Marker,
		pkt.PayloadType,
		pkt.SequenceNumber,
		pkt.Timestamp,
		pkt.SSRC,
		pkt.CSRC,
		pkt.ExtensionProfile,
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
