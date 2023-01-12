package video

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
)

type SampleWithTimingHint struct {
	Sample     media.Sample
	TimingHint time.Duration
}

type SamplingReader interface {
	NextSample() (SampleWithTimingHint, error)
}

func GetReader(reader io.Reader, logger hclog.Logger) (SamplingReader, error) {
	buf := make([]byte, 4)
	n, err := reader.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("Reader failed: %v", err)
	}
	if n < 4 {
		return nil, fmt.Errorf("Expected streaming reader header")
	}
	mediaVideoType := binary.LittleEndian.Uint32(buf)

	switch mediaVideoType {
	case uint32(MediaVideoTypeH264):
		logger.Info("reading video data from source", "format", "h264")
		return newH264SamplingReader(reader, logger)
	case uint32(MediaVideoTypeVP8):
		logger.Info("reading video data from source", "format", "vp8")
		return newVP8SamplingReader(reader)
	default:
		return nil, fmt.Errorf("Unknown media video type")
	}
}

func newH264SamplingReader(source io.Reader, logger hclog.Logger) (SamplingReader, error) {
	r, err := h264reader.NewReader(source)
	if err != nil {
		return nil, fmt.Errorf("Failed creating new H264 reader: %v", err)
	}
	return &h264SamplingReader{r: r, l: logger}, nil
}

type h264SamplingReader struct {
	r *h264reader.H264Reader
	l hclog.Logger
}

func (impl *h264SamplingReader) NextSample() (SampleWithTimingHint, error) {
	nal, nalErr := impl.r.NextNAL()
	if nalErr != nil {
		if errors.Is(nalErr, io.EOF) {
			return SampleWithTimingHint{}, io.EOF // always make sure we return EOF when we want EOF
		}
		return SampleWithTimingHint{}, nalErr
	}
	impl.l.Info("Delivering NAL", "length", len(nal.Data))
	return SampleWithTimingHint{
		Sample: media.Sample{
			Data:     nal.Data,
			Duration: time.Second,
		},
		TimingHint: time.Millisecond * 33, // 30 fps
	}, nil
}

func newVP8SamplingReader(source io.Reader) (SamplingReader, error) {
	r, h, err := ivfreader.NewWith(source)
	if err != nil {
		return nil, fmt.Errorf("Failed creating new VP8 reader: %v", err)
	}
	return &vp8SamplingReader{r: r, h: h}, nil
}

type vp8SamplingReader struct {
	r      *ivfreader.IVFReader
	h      *ivfreader.IVFFileHeader
	lastTs uint64
}

func (impl *vp8SamplingReader) NextSample() (SampleWithTimingHint, error) {
	frame, header, ivfErr := impl.r.ParseNextFrame()
	if ivfErr != nil {
		if errors.Is(ivfErr, io.EOF) {
			return SampleWithTimingHint{}, io.EOF // always make sure we return EOF when we want EOF
		}
		return SampleWithTimingHint{}, ivfErr
	}
	var diffMs int64
	if impl.lastTs > 0 {
		diff := header.Timestamp - impl.lastTs
		diffMs = int64(diff)
	}
	impl.lastTs = header.Timestamp
	return SampleWithTimingHint{
		Sample: media.Sample{
			Data:            frame,
			PacketTimestamp: uint32(header.Timestamp),
			Duration:        time.Second,
		},
		TimingHint: time.Duration(diffMs/100) * time.Millisecond,
	}, nil
}
