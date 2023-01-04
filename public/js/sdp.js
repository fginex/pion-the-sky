window.sdp = (topCallback) => {
    var f = (callback) => {
        pc = new RTCPeerConnection({
            iceServers: [
                {
                    urls: 'stun:stun.l.google.com:19302'
                }
            ]
        })
        userMediaSettings = { video: true, audio: true }
        navigator.mediaDevices.getUserMedia(userMediaSettings).then(stream => {
            var senders = []
            stream.getTracks().forEach(track => {
                senders.push(pc.addTrack(track, stream))
            })
            pc.createOffer().then(rtcsdp => callback(stream, pc, rtcsdp)).catch(log)
        }).catch(log)
    }
    f((stream, pc, rtcsdp) => {
        
        // disconnect media
        stream.getTracks().forEach((track) => track.stop());
        pc.close()

        var lines = rtcsdp.sdp.split(String.fromCharCode(10))

        var lookingUpVideoCodecs = false
        var videoCodecIds = []

        var lookingUpAudioCodecs = false
        var audioCodecIds = []

        var audioCodecs = {}
        var videoCodecs = {}
        var ensureCodec = (kind, id) => {
            var source = (() => {
                if (kind === "audio") {
                    return audioCodecs
                }
                if (kind === "video") {
                    return videoCodecs
                }
                return {}
            })()
            if (source[id] === undefined) {
                if (kind === "audio") {
                    source[id] = {
                        mimeType: "",
                        clockRate: 0,
                        channels: 0,
                        sdpFmtpLine: "",
                        rtcpFb: []
                    }
                } else if (kind === "video") {
                    source[id] = {
                        mimeType: "",
                        clockRate: 0,
                        sdpFmtpLine: "",
                        rtcpFb: []
                    }
                }
            }
            return source
        }

        for (var i=0; i<lines.length; i++) {
            var line = lines[i].trim()
            if (line.indexOf("m=video") == 0) {
                var parts = line.split(" ")
                parts.splice(0, 3)
                videoCodecIds = parts
                lookingUpVideoCodecs = true
                lookingUpAudioCodecs = false
            } else if (line.indexOf("m=audio") == 0) {
                var parts = line.split(" ")
                parts.splice(0, 3)
                audioCodecIds = parts
                lookingUpVideoCodecs = false
                lookingUpAudioCodecs = true
            } else {
                // Process audio codec data
                if (lookingUpAudioCodecs) {
                    if (line.indexOf("a=rtpmap:") === 0) {
                        var parts = line.split(" ")
                        var codecId = parts[0].replace("a=rtpmap:", "")
                        var codecData = parts[1].split("/")
                        if (window.arrayContains(audioCodecIds, codecId)) {
                            var container = ensureCodec("audio", codecId)
                            container[codecId].mimeType = `audio/${codecData.shift()}`
                            container[codecId].clockRate = parseInt(codecData.shift())
                            container[codecId].channels = (() => {
                                if (codecData.length === 1) {
                                    return parseInt(codecData.shift())
                                }
                                return 1
                            })()
                        }
                    } else if (line.indexOf("a=fmtp:") === 0) {
                        var parts = line.split(" ")
                        var codecId = parts[0].replace("a=fmtp:", "")
                        if (window.arrayContains(audioCodecIds, codecId)) {
                            var container = ensureCodec("audio", codecId)
                            container[codecId].sdpFmtpLine = parts[1]
                        }
                    } else if (line.indexOf("a=rtcp-fb:") === 0) {
                        var parts = line.split(" ")
                        var codecId = parts[0].replace("a=rtcp-fb:", "")
                        if (window.arrayContains(audioCodecIds, codecId)) {
                            var container = ensureCodec("audio", codecId)
                            container[codecId].rtcpFb.push(parts[1])
                        }
                    }
                }
                // Process video codec data
                if (lookingUpVideoCodecs) {
                    if (line.indexOf("a=rtpmap:") === 0) {
                        var parts = line.split(" ")
                        var codecId = parts[0].replace("a=rtpmap:", "")
                        var codecData = parts[1].split("/")
                        if (window.arrayContains(videoCodecIds, codecId)) {
                            var container = ensureCodec("video", codecId)
                            container[codecId].mimeType = `video/${codecData.shift()}`
                            container[codecId].clockRate = parseInt(codecData.shift())
                        }
                    } else if (line.indexOf("a=fmtp:") === 0) {
                        var parts = line.split(" ")
                        var codecId = parts[0].replace("a=fmtp:", "")
                        if (window.arrayContains(videoCodecIds, codecId)) {
                            var container = ensureCodec("video", codecId)
                            container[codecId].sdpFmtpLine = parts[1]
                        }
                    } else if (line.indexOf("a=rtcp-fb:") === 0) {
                        var parts = line.split(" ")
                        var codecId = parts[0].replace("a=rtcp-fb:", "")
                        if (window.arrayContains(videoCodecIds, codecId)) {
                            var container = ensureCodec("video", codecId)
                            container[codecId].rtcpFb.push(parts[1])
                        }
                    }
                }
            }
        }
        topCallback(audioCodecs, videoCodecs)
    })
}
