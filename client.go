package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc"
	"github.com/pion/webrtc/pkg/media/rtpdump"
)

// PeerClientType represents the types of signal messages
type PeerClientType int

const (
	// PctUndecided - undetermined
	PctUndecided = PeerClientType(iota)

	// PctRecord - recording client
	PctRecord = PeerClientType(iota)

	// PctPlayback - playback client
	PctPlayback = PeerClientType(iota)
)

// PeerClient represents a server-side client used as a peer to the browser client.
type PeerClient struct {
	id            string
	ct            PeerClientType
	pc            *webrtc.PeerConnection
	ws            *websocket.Conn
	localSession  string
	remoteSession string

	services *WebRTCService

	audioBuf *bytes.Buffer
	videoBuf *bytes.Buffer

	closeCh chan struct{}

	wg    sync.WaitGroup
	mutex sync.Mutex
}

// CreateNewPeerClient creates a new server peer client.
func CreateNewPeerClient(conn *websocket.Conn, services *WebRTCService) (*PeerClient, error) {

	client := PeerClient{
		id:      guuid.New().String(),
		ct:      PctUndecided,
		ws:      conn,
		closeCh: make(chan struct{}),

		audioBuf: bytes.NewBuffer([]byte{}),
		videoBuf: bytes.NewBuffer([]byte{}),

		services: services,
	}

	log.Printf("Server Peer Client %s created.\n", client.id)

	go client.eventLoop()

	return &client, nil
}

// Close - closes a client's peer and signal connections.
func (c *PeerClient) Close() {
	if c.IsClosed() {
		return
	}

	c.mutex.Lock()
	close(c.closeCh)
	c.mutex.Unlock()

	c.ws.Close()

	if c.pc != nil {
		c.pc.Close()
	}

	c.wg.Wait()

	if c.ct == PctRecord {
		c.services.SaveVideo(c.id, c.videoBuf)
	}

	log.Printf("Client %s closed.\n", c.id)
}

func (c *PeerClient) eventLoop() {
	c.wg.Add(1)
	defer func() {
		log.Printf("Server Peer Client %s is exiting its event loop.\n", c.id)
		c.wg.Done()
	}()

	var ev SignalMessage
	var err error

	for {
		if c.IsClosed() {
			return
		}

		err = c.ws.ReadJSON(&ev)
		if err != nil {
			log.Printf("Client %s error %s\n", c.id, err)
			go c.Close()
			return
		}

		err = ev.Unmarshal()
		if err != nil {
			log.Printf("Client %s error receiving signal event %s\n", c.id, err)
			continue
		}

		log.Printf("Client %s received event: %s\n", c.id, ev.Op)

		switch ev.id {
		case SmRecord:
			if c.ct != PctUndecided {
				c.sendError("Peer client is already either recording or playing. Please disconnect and try again.")
				continue
			}
			c.ct = PctRecord
			c.localSession = ev.Data
			go func() {
				c.wg.Add(1)
				defer c.wg.Done()
				err = c.services.CreateRecordingConnection(c)
				if err != nil {
					log.Printf("Client %s error %s\n", c.id, err)
				}
			}()

		case SmPlay:
			if c.ct != PctUndecided {
				c.sendError("Peer client is already either recording or playing. Please disconnect and try again.")
				continue
			}
			c.ct = PctPlayback
			c.localSession = ev.Data
			go func() {
				c.wg.Add(1)
				defer c.wg.Done()
				err = c.services.CreatePlaybackConnection(c)
				if err != nil {
					log.Printf("Client %s error %s\n", c.id, err)
				}
			}()
		}
	}
}

// IsClosed checks to see if this client has been shutdown
func (c *PeerClient) IsClosed() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	select {
	case _, ok := <-c.closeCh:
		if !ok {
			return true
		}
	default:
	}
	return false
}

// startRemoteSession - Completes the session initiation with the client.
func (c *PeerClient) startRemoteSession() error {

	offer := webrtc.SessionDescription{}
	Decode(c.localSession, &offer)

	// Set the remote session description
	err := c.pc.SetRemoteDescription(offer)
	if err != nil {
		return err
	}

	// Create answer
	answer, err := c.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	// Starts the UDP listeners
	err = c.pc.SetLocalDescription(answer)
	if err != nil {
		return err
	}

	// Send back the answer (this peer's session description) in base64 to the browser client
	c.remoteSession = Encode(answer)

	msg := SignalMessage{}
	msg.id = SmAnswer
	msg.Data = c.remoteSession
	msg.Marshal()

	err = c.ws.WriteJSON(&msg)
	if err != nil {
		return err
	}
	return nil
}

// recordTrack records raw audio and video packets off the given track
func (c *PeerClient) recordTrack(track *webrtc.Track) error {
	codec := track.Codec()

	c.wg.Add(1)
	defer func() {
		log.Printf("Record track loop for %s exiting client id:%s\n", codec.Name, c.id)
		c.wg.Done()
	}()

	log.Printf("Recording %s track for client id:%s\n", codec.Name, c.id)

	header := rtpdump.Header{
		Start:  time.Unix(9, 0).UTC(),
		Source: net.IPv4(2, 2, 2, 2),
		Port:   2222,
	}

	var writer *rtpdump.Writer
	var err error

	if codec.Name == webrtc.Opus {
		fmt.Printf("recordTrack client %s Opus Audio track 48 kHz, 2 channels\n", c.id)
		writer, err = rtpdump.NewWriter(c.audioBuf, header)
	} else if codec.Name == webrtc.VP8 {
		fmt.Printf("recordTrack client %s VP8 Video track\n", c.id)
		writer, err = rtpdump.NewWriter(c.videoBuf, header)
	} else {
		panic(fmt.Sprintf("recordTrack unexpected codec %s", codec.Name))
	}

	if err != nil {
		return err
	}

	for {
		if c.IsClosed() {
			break
		}

		rtpPacket, err := track.ReadRTP()
		if err != nil {
			return err
		}

		raw, _ := rtpPacket.Marshal()

		dpacket := rtpdump.Packet{
			Offset:  0,
			IsRTCP:  false,
			Payload: raw,
		}

		writer.WritePacket(dpacket)
	}

	return nil
}

// streamVideoToTrack records raw audio and video packets off the given track
func (c *PeerClient) streamVideoToTrack(outputTrack *webrtc.Track) {
	codec := outputTrack.Codec()
	ticker := time.NewTicker(time.Duration(40) * time.Second / time.Duration(1000))

	c.wg.Add(1)
	defer func() {
		ticker.Stop()
		log.Printf("StreamTo track loop for %s exiting client id:%s\n", codec.Name, c.id)
		c.wg.Done()
	}()

	rtp := rtp.Packet{}
	seq := uint16(100)
	tsbegin := uint32(0)
	tsmod := uint32(0)
	tsprev := uint32(0)
	tsdelta := uint32(0)

	for { // Loop thru the video clips
		if c.IsClosed() {
			return
		}

		// NOTE: The video clips will not be played back in order since we are just
		// iterating thru a map.

		for id, buf := range c.services.Videos { //TODO: Note: not thread safe
			if c.IsClosed() {
				return
			}

			log.Printf("Started streaming %s to Client %s...\n", id, c.id)

			r, _, err := rtpdump.NewReader(bytes.NewReader(buf.Bytes()))
			if err != nil {
				log.Println(err)
				return
			}

			clipreset := true

			for range ticker.C {
				if c.IsClosed() {
					return
				}

				pkt, err := r.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Println(err)
					return
				}

				rtp.Unmarshal(pkt.Payload)

				// ---
				// NOTE: You can alter the packets here for testing.
				// ---

				rtp.SSRC = outputTrack.SSRC()
				rtp.PayloadType = webrtc.DefaultPayloadTypeVP8

				// Adjust the timestamp and sequence for streaming
				if clipreset {
					clipreset = false
					tsdelta = 0
				} else {
					tsdelta = rtp.Timestamp - tsprev
				}
				tsprev = rtp.Timestamp

				if tsbegin == 0 {
					tsbegin = 1
					tsmod = tsbegin
				} else {
					tsmod = tsmod + tsdelta
				}
				rtp.SequenceNumber = seq
				seq++
				rtp.Timestamp = tsmod

				err = outputTrack.WriteRTP(&rtp)
				if err != nil {
					log.Println(err)
					return
				}
			}
			log.Printf("Finished streaming %s to Client %s...\n", id, c.id)
		}
	}
}

func (c *PeerClient) sendError(errMsg string) error {
	log.Printf("Client %s sending error to peer: %s\n", c.id, errMsg)

	msg := SignalMessage{}
	msg.id = SmError
	msg.Data = errMsg
	msg.Marshal()

	err := c.ws.WriteJSON(&msg)
	if err != nil {
		return err
	}

	return nil
}
