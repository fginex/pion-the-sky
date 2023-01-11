export interface MediaResolveConfiguration {
    iceServers: string[]
    mediaStreamConstraints?: MediaStreamConstraints
}

export interface MediaError {
    reason: string
    cause?: any
}

export class ConnectedPeerClient {

    private peerConn: RTCPeerConnection
    private config: MediaResolveConfiguration

    private stream: MediaStream | null = null

    private inProgressCandidate: Promise<void> | null = null
    private inProgressCandidateResolve: (value: void | PromiseLike<void>) => void = (_: void | PromiseLike<void>) => {}
    private inProgressCandidateReject: (reason?: any) => void = (_?: any) => {}

    private audioTarget: HTMLAudioElement | null | undefined = undefined
    private videoTarget: HTMLVideoElement | null | undefined = undefined

    constructor(pc: RTCPeerConnection, config: MediaResolveConfiguration) {
        this.peerConn = pc
        this.config = config
        this.peerConn.onconnectionstatechange = this.onPeerConnectionStateChange.bind(this)
        this.peerConn.onicecandidate = this.onPeerConnectionIceCandidate.bind(this)
        this.peerConn.onicecandidateerror = this.onPeerConnectionIceCandidateError.bind(this)
        this.peerConn.oniceconnectionstatechange = this.onPeerConnectionIceConnectionStateChange.bind(this)
        this.peerConn.onicegatheringstatechange = this.onPeerIceGatheringStateChange.bind(this)
        this.peerConn.onnegotiationneeded = this.onPeerNegotiationNeeded.bind(this)
        this.peerConn.onsignalingstatechange = this.onPeerConnectionSignalingStateChange.bind(this)
        this.peerConn.ontrack = this.onPeerConnectionTrack.bind(this)
    }

    connectMedia(audioTarget?: HTMLAudioElement | null, videoTarget?: HTMLVideoElement | null) {
        const $self = this
        return new Promise<MediaStream>((resolve, reject) => {
            navigator.mediaDevices
                .getUserMedia(getMediaConstraintsOrDefault(this.config.mediaStreamConstraints))
                    .then(stream => {
                        stream.getTracks().forEach(function (track) {
                            const sender = $self.peerConn.addTrack(track, stream)
                            $self.peerConn.addTransceiver(track.kind, { direction: 'sendrecv' })
                        })

                        stream.getVideoTracks().forEach(track => {
                            console.log("Video track settings: ", track.getCapabilities(), track.getConstraints(), track.getSettings(), track)
                        })
                        stream.getAudioTracks().forEach(track => {
                            console.log("Audio track settings: ", track.getCapabilities(), track.getConstraints(), track.getSettings(), track)
                        })

                        $self.peerConn.createOffer()
                            .then(d => {
                                $self.stream = stream
                                $self.peerConn.setLocalDescription(d)
                                    .then(() => {
                                        resolve(stream)
                                            $self.audioTarget = audioTarget
                                            $self.videoTarget = videoTarget
                                    })
                                    .catch(err => {
                                        $self.disconnectMedia()
                                        const e: MediaError = {
                                            reason: "Failed updating local session description",
                                            cause: err,
                                        }
                                        reject(e)
                                    })
                            })
                            .catch(err => {
                                $self.disconnectMedia()
                                const e: MediaError = {
                                    reason: "Failed creating media offer",
                                    cause: err,
                                }
                                reject(e)
                            })
                    })
                    .catch(err => {
                        const e: MediaError = {
                            reason: "Failed resolving user media",
                            cause: err,
                        }
                        reject(e)
                    })
        })
    }

    disconnectMedia() {
        if (this.stream !== null) {
            const $stream = this.stream
            return new Promise<void>((resolve, _) => {
                $stream.getTracks().forEach(track => track.stop())
                resolve()
            })
        }
        return new Promise<void>((_, reject) => {
            const e: MediaError = {
                reason: "Media not connected",
            }
            reject(e)
        })
    }

    withCandidate() {
        this.inProgressCandidate = new Promise<void>((resolve, reject) => {
            this.inProgressCandidateResolve = resolve.bind(this)
            this.inProgressCandidateReject = reject.bind(this)
        })
        return this.inProgressCandidate
    }

    localDescription() {
        return this.peerConn.localDescription
    }

    setRemoteDescription(rsdp: RTCSessionDescription) {
        this.peerConn.setRemoteDescription(rsdp)
    }

    // PeerConnection callbacks:
    private onPeerConnectionStateChange(ev: Event) {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/connectionstatechange_event
        console.log("Peer connection: connection state change", ev)
    }

    private onPeerConnectionIceCandidate(ev: RTCPeerConnectionIceEvent) {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/icecandidate_event
        if (ev.candidate !== null) {
            if (this.inProgressCandidate !== null) {
                this.inProgressCandidateResolve()
            }
            console.log("Peer connection: new ICE candidate", ev)
        }
    }

    private onPeerConnectionIceCandidateError(ev: Event) {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/icecandidateerror_event
        console.log("Peer connection: new ICE candidate error", ev)
    }

    private onPeerConnectionIceConnectionStateChange(ev: Event) {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/iceconnectionstatechange_event
        if (this.peerConn !== null) {
            console.log("Peer connection: ICE connection state change", ev, this.peerConn.iceConnectionState)
        }
    }

    private onPeerIceGatheringStateChange(ev: Event) {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/icegatheringstatechange_event
        if (this.peerConn !== null) {
            console.log("Peer connection: ICE gathering state change", ev, this.peerConn.iceGatheringState)
        }
    }

    private onPeerNegotiationNeeded(ev: Event) {
        // https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/negotiationneeded_event
        console.log("Peer connection: negotiation needed", ev)
    }

    private onPeerConnectionSignalingStateChange(ev: Event) {
        console.log("Peer connection: signaling state change", ev)
    }

    private onPeerConnectionTrack(ev: RTCTrackEvent) {
        console.log("Peer connection: received a track:", ev)
        if (ev.track.kind === "audio") {
            if (this.audioTarget !== undefined) {
                if (this.audioTarget !== null) {
                    const $a = this.audioTarget
                    console.log("Received ", ev.streams.length, " audio streams")
                    //ev.streams.forEach(stream => $a.srcObject = stream)
                }
            }
        }
        if (ev.track.kind === "video") {
            if (this.videoTarget !== undefined) {
                if (this.videoTarget !== null) {
                    const $v = this.videoTarget
                    console.log("Received ", ev.streams.length, " video streams")
                    ev.streams.forEach(stream => $v.srcObject = stream)
                }
            }
        }
    }

}

export const peerConnectionWithMedia = (config: MediaResolveConfiguration): Promise<ConnectedPeerClient> => {
    return new Promise<ConnectedPeerClient>((resolve, reject) => {
        try {
            const rtcConfig: RTCConfiguration = {}
            if (config.iceServers.length > 0) {
                rtcConfig.iceServers = [{
                    urls: config.iceServers,
                }]
            }
            const pc = new RTCPeerConnection(rtcConfig)
            resolve(new ConnectedPeerClient(pc, config))
        } catch (err) {
            const e: MediaError = {
                reason: "Failed creating peer client",
                cause: err,
            }
            reject(e)
        }
    })
}

const getMediaConstraintsOrDefault = (mediaStreamConstraints?: MediaStreamConstraints): MediaStreamConstraints => {
    if (mediaStreamConstraints !== undefined) {
        return mediaStreamConstraints
    }
    //return {video: true, audio: true}
    return {video: {frameRate: {max: 30}}, audio: true}
}
