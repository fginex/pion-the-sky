package codecs

import "github.com/pion/webrtc/v3"

const mimeTypeVideoRtx = "video/rtx"
const enableH264 = false

// AudioCodecs returns a list of audio codecs we support.
func AudioCodecs() []webrtc.RTPCodecParameters {
	return []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     webrtc.MimeTypeOpus,
				ClockRate:    48000,
				Channels:     2,
				SDPFmtpLine:  "minptime=10;useinbandfec=1",
				RTCPFeedback: nil,
			},
			PayloadType: 111,
		},
	}
}

// VideoCodecs returns a list of audio codecs we support.
func VideoCodecs() []webrtc.RTPCodecParameters {

	videoRTCPFeedback := []webrtc.RTCPFeedback{
		{Type: "goog-remb", Parameter: ""},
		{Type: "ccm", Parameter: "fir"},
		{Type: "nack", Parameter: ""},
		{Type: "nack", Parameter: "pli"},
	}

	codecs := []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     webrtc.MimeTypeVP8,
				ClockRate:    90000,
				Channels:     0,
				SDPFmtpLine:  "max-fs=12288;max-fr=30",
				RTCPFeedback: videoRTCPFeedback,
			},
			PayloadType: 96,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     "video/rtx",
				ClockRate:    90000,
				Channels:     0,
				SDPFmtpLine:  "apt=96",
				RTCPFeedback: nil,
			},
			PayloadType: 97,
		},
	}

	if enableH264 {

		// Leaving this for a reference but...
		// With Firefox, the webrtc peer in this program complains that the remote does not support the codec.
		// With Safari, the client does not complain but when streaming the H264 data back, no video nor audio is rendered.
		// TODO: revisit later...

		videoRTCPH264Feedback := []webrtc.RTCPFeedback{
			{Type: "goog-remb", Parameter: ""},
			{Type: "ccm", Parameter: "fir"},
			{Type: "nack", Parameter: ""},
			{Type: "nack", Parameter: "pli"},
			{Type: "transport-cc", Parameter: ""},
		}
		codecs = append(codecs, []webrtc.RTPCodecParameters{
			{
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:     webrtc.MimeTypeH264,
					ClockRate:    90000,
					Channels:     0,
					SDPFmtpLine:  "profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1",
					RTCPFeedback: videoRTCPH264Feedback,
				},
				PayloadType: 126,
			},
			{
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:     mimeTypeVideoRtx,
					ClockRate:    90000,
					Channels:     0,
					SDPFmtpLine:  "apt=126",
					RTCPFeedback: nil,
				},
				PayloadType: 127,
			},
		}...)
	}

	return codecs
}
