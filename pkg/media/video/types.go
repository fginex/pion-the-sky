package video

import "encoding/binary"

type MediaVideoType uint32

const (
	MediaVideoTypeH264 MediaVideoType = 1
	MediaVideoTypeVP8  MediaVideoType = 2
)

func HeaderH264() []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(MediaVideoTypeH264))
	return buf
}

func HeaderVP8() []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(MediaVideoTypeVP8))
	return buf
}
