enum CodecType {
    Audio = "audio",
    Video = "video",
}

export interface AudioCodec {
    mimeType: string,
    clockRate: number,
    channels: number,
    sdpFmtpLine: string,
    rtcpFb: string[]
}

export interface VideoCodec {
    mimeType: string,
    clockRate: number,
    sdpFmtpLine: string,
    rtcpFb: string[]
}

export interface CodecsReadError {
    reason: string
}

export interface Codecs {
    audio: {[key: string]: AudioCodec}
    video: {[key: string]: VideoCodec}
}

export const readCodecs = (sessionDescription: RTCSessionDescription): Promise<Codecs> => {
    return new Promise<Codecs>((resolve, reject) => {
        const response: Codecs = {
            audio: {},
            video: {}
        }

        let lookingUpVideoCodecs = false
        let videoCodecIds: string[] = []
        let lookingUpAudioCodecs = false
        let audioCodecIds: string[] = []

        try {

            var lines = sessionDescription.sdp.split(String.fromCharCode(10))

            // The code below looks a bit messy but it's single pass.
            // It relies on the fact that the SDP provides the somewhat-ordered information.
            // It's audio and video data, can be in any order
            // but the list of codecs follow that specific line.

            for (var i=0; i<lines.length; i++) {
                var line = lines[i].trim()
                if (line.indexOf("m=video") == 0) {
                    const parts = line.split(" ")
                    parts.splice(0, 3)
                    if (parts.length < 4) {
                        return rejectWithError(`invalid m=video line, expected 'm=video port protocol ids', received ${line}`, reject)
                    }
                    videoCodecIds = parts
                    lookingUpVideoCodecs = true
                    lookingUpAudioCodecs = false
                } else if (line.indexOf("m=audio") == 0) {
                    const parts = line.split(" ")
                    if (parts.length < 4) {
                        return rejectWithError(`invalid m=audio line, expected 'm=audio port protocol ids', received ${line}`, reject)
                    }
                    parts.splice(0, 3)
                    audioCodecIds = parts
                    lookingUpVideoCodecs = false
                    lookingUpAudioCodecs = true
                } else {

                    if (lookingUpAudioCodecs) { // Process audio codec data
                        if (line.indexOf("a=rtpmap:") === 0) {
                            const parts = line.split(" ")
                            const codecId = parts[0].replace("a=rtpmap:", "")
                            const codecData = parts[1].split("/")
                            if (arrayContains(audioCodecIds, codecId)) {
                                ensureAudioCodec(codecId, response)
                                let [name, clockRate, channels] = codecData
                                response.audio[codecId].mimeType = `audio/${name}`
                                response.audio[codecId].clockRate = parseInt(clockRate)
                                response.audio[codecId].channels = (channels !== undefined) ? parseInt(channels) : 1
                            }
                        } else if (line.indexOf("a=fmtp:") === 0) {
                            var parts = line.split(" ")
                            var codecId = parts[0].replace("a=fmtp:", "")
                            if (arrayContains(audioCodecIds, codecId)) {
                                ensureAudioCodec(codecId, response)
                                response.audio[codecId].sdpFmtpLine = parts[1]
                            }
                        } else if (line.indexOf("a=rtcp-fb:") === 0) {
                            var parts = line.split(" ")
                            var codecId = parts[0].replace("a=rtcp-fb:", "")
                            if (arrayContains(audioCodecIds, codecId)) {
                                ensureAudioCodec(codecId, response)
                                response.audio[codecId].rtcpFb.push(parts[1])
                            }
                        }
                    } // end audio codecs lookup

                    if (lookingUpVideoCodecs) { // Process video codec data
                        if (line.indexOf("a=rtpmap:") === 0) {
                            var parts = line.split(" ")
                            var codecId = parts[0].replace("a=rtpmap:", "")
                            var codecData = parts[1].split("/")
                            if (arrayContains(videoCodecIds, codecId)) {
                                ensureVideoCodec(codecId, response)
                                let [name, clockRate] = codecData
                                response.video[codecId].mimeType = `video/${name}`
                                response.video[codecId].clockRate = parseInt(clockRate)
                            }
                        } else if (line.indexOf("a=fmtp:") === 0) {
                            var parts = line.split(" ")
                            var codecId = parts[0].replace("a=fmtp:", "")
                            if (arrayContains(videoCodecIds, codecId)) {
                                ensureVideoCodec(codecId, response)
                                response.video[codecId].sdpFmtpLine = parts[1]
                            }
                        } else if (line.indexOf("a=rtcp-fb:") === 0) {
                            var parts = line.split(" ")
                            var codecId = parts[0].replace("a=rtcp-fb:", "")
                            if (arrayContains(videoCodecIds, codecId)) {
                                ensureVideoCodec(codecId, response)
                                response.video[codecId].rtcpFb.push(parts[1])
                            }
                        }
                    } // end video codecs lookup

                }
            } // end for
            
            resolve(response)

        } catch (e) {
            reject(e)
        }

    })
}

const rejectWithError = (message: string, reject: (reason?: any) => void) => {
    const e: CodecsReadError = {
        reason: message
    }
    reject(e)
}

const ensureAudioCodec = (id: string, codecs: Codecs) => {
    if (!arrayContains(Object.keys(codecs.audio), id)) {
        const newCodec: AudioCodec = {
            mimeType: "",
            clockRate: 0,
            channels: 0,
            sdpFmtpLine: "",
            rtcpFb: [],
        }
        codecs.audio[id] = newCodec
    }
}

const ensureVideoCodec = (id: string, codecs: Codecs) => {
    if (!arrayContains(Object.keys(codecs.video), id)) {
        const newCodec: VideoCodec = {
            mimeType: "",
            clockRate: 0,
            sdpFmtpLine: "",
            rtcpFb: [],
        }
        codecs.video[id] = newCodec
    }
}

const arrayContains = <T>(arr: T[], item: T): boolean => {
    for (var i=0; i<arr.length; i++) {
        if (arr[i] === item) {
            return true
        }
    }
    return false
}