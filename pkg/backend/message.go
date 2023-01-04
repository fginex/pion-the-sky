package backend

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

// SignalMessageType represents the types of signal messages
type SignalMessageType int

// SignalMessageType describes the signal message types
var signalMessageOps = [...]string{
	"UNDEFINED",
	"RECORD",
	"ANSWER",
	"PLAY",
	"ERROR",
}

const (
	// SmUndefined - undefined value
	SmUndefined = SignalMessageType(iota)

	// SmRecord - browser client sends local browser session description to server and server initiates recording
	SmRecord

	// SmAnswer - server responds with remote peer description
	SmAnswer

	// SmPlay - browser client sends to server to start streaming back the recorded video
	SmPlay

	// SmError - error
	SmError
)

// String - returns the string value
func (t SignalMessageType) String() string {
	return signalMessageOps[t]
}

// SignalMessage represents the format of a signal message over the websocket
type SignalMessage struct {
	id   SignalMessageType
	Op   string `json:"op"`
	Data string `json:"data"`
}

// Marshal populates the op field from the id field
func (t *SignalMessage) Marshal() {
	t.Op = t.id.String()
}

// Unmarshal sets the id field from the op field
func (t *SignalMessage) Unmarshal() error {

	for i, op := range signalMessageOps {
		if op == t.Op {
			t.id = SignalMessageType(i)
			return nil
		}
	}
	t.id = SmUndefined
	return fmt.Errorf("undefined SignalMessageType Op value %s", t.Op)
}

// NewSignalMessage creates a new message ready to be transported over the websocket.
func NewSignalMessage(t SignalMessageType, data string) *SignalMessage {
	return &SignalMessage{Op: t.String(), Data: data}
}

// MustReadStdin blocks until input is received from stdin
func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				panic(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return in
}

// Encode encodes the input in base64
// It can optionally zip the input before encoding
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	if compress {
		b = zip(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if compress {
		b = unzip(b)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func zip(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		panic(err)
	}
	err = gz.Flush()
	if err != nil {
		panic(err)
	}
	err = gz.Close()
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func unzip(in []byte) []byte {
	var b bytes.Buffer
	_, err := b.Write(in)
	if err != nil {
		panic(err)
	}
	r, err := gzip.NewReader(&b)
	if err != nil {
		panic(err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return res
}
