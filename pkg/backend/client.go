package backend

import (
	"bytes"
	"fmt"
	"sync"

	guuid "github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/pion/sdp/v2"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
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
	id string
	ct PeerClientType
	pc *webrtc.PeerConnection
	ws *websocket.Conn

	browserSD string
	serverSD  string

	sdParsed sdp.SessionDescription

	services *WebRTCService

	audioBuf *bytes.Buffer
	videoBuf *bytes.Buffer

	logger hclog.Logger

	closeCh chan struct{}

	wg    sync.WaitGroup
	mutex sync.Mutex
}

// CreateNewPeerClient creates a new server peer client.
func CreateNewPeerClient(conn *websocket.Conn, services *WebRTCService, logger hclog.Logger) (*PeerClient, error) {
	clientID := guuid.New().String()
	client := PeerClient{
		id:       clientID,
		ct:       PctUndecided,
		ws:       conn,
		closeCh:  make(chan struct{}),
		audioBuf: bytes.NewBuffer([]byte{}),
		videoBuf: bytes.NewBuffer([]byte{}),
		logger:   logger.Named(fmt.Sprintf("peer-%s", clientID)).With("client-id", clientID),
		services: services,
	}
	client.logger.Info("Server Peer Client created")
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
		c.services.SaveAudio(c.id, c.audioBuf)
	}

	c.logger.Info("Client closed")
}

func (c *PeerClient) eventLoop() {
	c.wg.Add(1)
	defer func() {
		c.logger.Info("Server Peer Client is exiting event loop")
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
			c.logger.Error("Client failed reading JSON data from WebSocket", "reason", err)
			go c.Close()
			return
		}

		err = ev.Unmarshal()
		if err != nil {
			c.logger.Error("Client failed receiving signal event", "reason", err)
			continue
		}

		c.logger.Info("Client received an event", "event", ev.Op)

		switch ev.id {
		case SmRecord:
			if c.ct != PctUndecided {
				c.sendError("Peer client is already either recording or playing. Please disconnect and try again.")
				continue
			}
			c.ct = PctRecord
			c.browserSD = ev.Data
			go func() {
				c.wg.Add(1)
				defer c.wg.Done()
				err = c.services.CreateRecordingConnection(c)
				if err != nil {
					c.logger.Error("Client recording error", "reason", err)
				}
			}()

		case SmPlay:
			if c.ct != PctUndecided {
				c.sendError("Peer client is already either recording or playing. Please disconnect and try again.")
				continue
			}
			if !c.services.HasRecordings() {
				c.sendError("There are no recorded videos to playback. Please record a video first.")
				continue
			}
			c.ct = PctPlayback
			c.browserSD = ev.Data
			go func() {
				c.wg.Add(1)
				defer c.wg.Done()
				err = c.services.CreatePlaybackConnection(c)
				if err != nil {
					c.logger.Error("Client playback error", "reason", err)
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

// startServerSession - Completes the session initiation with the client.
func (c *PeerClient) startServerSession() error {

	offer := webrtc.SessionDescription{}
	Decode(c.browserSD, &offer)

	// Some browser codec mappings might not match what
	// pion has. The work around is to pull the payload type
	// from the browser session description and modify the
	// streaming rtp packets accordingly. See this issue for
	// more details: https://github.com/pion/webrtc/issues/716
	err := c.sdParsed.Unmarshal([]byte(offer.SDP))
	if err != nil {
		return err
	}
	/*
		TODO: fix
			codec := sdp.Codec{
				Name: c.services.vc.MimeType,
			}
			c.pt, err = c.sdParsed.GetPayloadTypeForCodec(codec)
			if err != nil {
				return err
			}
	*/
	// ---

	// Set the remote session description
	err = c.pc.SetRemoteDescription(offer)
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

	// Send back the answer (this peer's session description) in base64 to the browser client.
	// Note modifications may be made to account for known issues. See ModServerSessionDescription()
	// for more details.
	c.serverSD = Encode(ModAnswer(&answer))

	msg := SignalMessage{}
	msg.id = SmAnswer
	msg.Data = c.serverSD
	msg.Marshal()

	err = c.ws.WriteJSON(&msg)
	if err != nil {
		return err
	}
	return nil
}

func (c *PeerClient) recordTrack(writer media.Writer, track *webrtc.TrackRemote) error {
	defer func() {
		writer.Close()
	}()
	for {
		if c.IsClosed() {
			break
		}
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			return err
		}
		if err := writer.WriteRTP(rtpPacket); err != nil {
			return err
		}
	}
	return nil
}

func (c *PeerClient) recordAudioTrack(track *webrtc.TrackRemote) error {
	audiowriter, err := oggwriter.NewWith(c.audioBuf, 48000, 2)
	if err != nil {
		return err
	}
	return c.recordTrack(audiowriter, track)
}

func (c *PeerClient) recordVideoTrack(track *webrtc.TrackRemote) error {
	videowriter, err := ivfwriter.NewWith(c.videoBuf)
	if err != nil {
		return err
	}
	return c.recordTrack(videowriter, track)
}

func (c *PeerClient) sendError(errMsg string) error {
	c.logger.Error("Client reporting error to a peer", "error", errMsg)

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
