import { Buffer } from 'buffer'

export const rtcSessionDescriptionToBase64 = (sdp: RTCSessionDescription): string => {
    return Buffer.from(JSON.stringify(sdp)).toString("base64")
}