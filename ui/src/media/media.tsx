import { useEffect, useState } from "react"
import { Buffer } from 'buffer'

export type RTCSessionDescriptionAsString = () => string

export interface MediaClient {
    startMedia: (iceServers: string[]) => void
}

export interface MediaError {
    reason: string | Event
    cause?: any
}

export interface MediaProps {
    onICECandidate: (candidate: RTCIceCandidate,
        localSDP: RTCSessionDescription,
        localSDPAsString: RTCSessionDescriptionAsString) => void
    onMediaClient: (client: MediaClient) => void
    onMediaError: (reason: MediaError) => void
}

export const Media = (props: MediaProps) => {

    const [peerConnection, setPeerConnection] = useState<RTCPeerConnection | null>(null)

    const startMedia = (iceServers: string[]) => {

        try {
            let pc = new RTCPeerConnection({iceServers: [{
                urls: iceServers,
            }]})

            navigator.mediaDevices.getUserMedia({video: true, audio: true}).
                then(stream => {
                    // do I really have to call createOffer before stream getTracks()?
                    stream.getTracks().forEach(function (track) {
                        pc.addTrack(track, stream)
                    })
                    pc.createOffer()
                        .then(d => {
                            pc.setLocalDescription(d)
                            setPeerConnection(pc)
                        })
                        .catch(e => props.onMediaError({reason: "Failed creating offer", cause: e}))
                }).catch(e => {
                    props.onMediaError({reason: "Failed creating offer", cause: e})
                })

        } catch (e: any) {
            return
        }
    }

    const onPeerConnectionStateChange = (ev: Event) => {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/connectionstatechange_event
        console.log("Peer connection: connection state change", ev)
    }

    const onPeerConnectionIceCandidate = (ev: RTCPeerConnectionIceEvent) => {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/icecandidate_event
        console.log("Peer connection: new ICE candidate", ev)
        if (ev.candidate !== null && peerConnection !== null && peerConnection.localDescription !== null) {
            props.onICECandidate(ev.candidate, peerConnection.localDescription, (): string => {
                return Buffer.from(JSON.stringify(peerConnection.localDescription)).toString("base64")
            })
            console.log("Local session description ready. Ready to play streams back when you are.")
        }
    }

    const onPeerConnectionIceCandidateError = (ev: Event) => {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/icecandidateerror_event
        console.log("Peer connection: new ICE candidate error", ev)
    }

    const onPeerConnectionIceConnectionStateChange = (ev: Event) => {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/iceconnectionstatechange_event
        if (peerConnection !== null) {
            console.log("Peer connection: ICE connection state change", ev, peerConnection.iceConnectionState)
        }
    }

    const onPeerIceGatheringStateChange = (ev: Event) => {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/icegatheringstatechange_event
        if (peerConnection !== null) {
            console.log("Peer connection: ICE gathering state change", ev, peerConnection.iceGatheringState)
        }
    }

    const onPeerNegotiationNeeded = (ev: Event) => {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/negotiationneeded_event
        console.log("Peer connection: negotiation needed", ev)
    }

    const onPeerConnectionSignalingStateChange = (ev: Event) => {
        console.log("Peer connection: signaling state change", ev)
    }

    useEffect(() => {
        if (peerConnection !== null) {
            peerConnection.onconnectionstatechange = onPeerConnectionStateChange
            peerConnection.onicecandidate = onPeerConnectionIceCandidate
            peerConnection.onicecandidateerror = onPeerConnectionIceCandidateError
            peerConnection.oniceconnectionstatechange = onPeerConnectionIceConnectionStateChange
            peerConnection.onicegatheringstatechange = onPeerIceGatheringStateChange
            peerConnection.onnegotiationneeded = onPeerNegotiationNeeded
            peerConnection.onsignalingstatechange = onPeerConnectionSignalingStateChange
        } else {
            props.onMediaClient({startMedia: startMedia})
        }
    }, [peerConnection])

    return (<></>)
}