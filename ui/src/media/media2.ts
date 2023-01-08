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

    constructor(pc: RTCPeerConnection, config: MediaResolveConfiguration) {
        this.peerConn = pc
        this.config = config
    }

    connectMedia() {
        const $self = this
        return new Promise<void>((resolve, reject) => {
            navigator.mediaDevices
                .getUserMedia(getMediaConstraintsOrDefault(this.config.mediaStreamConstraints))
                    .then(stream => {
                        stream.getTracks().forEach(function (track) {
                            $self.peerConn.addTrack(track, stream)
                        })
                        $self.peerConn.createOffer()
                            .then(d => {
                                $self.stream = stream
                                $self.peerConn.setLocalDescription(d)
                                    .then(() => resolve())
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

    localDescription() {
        return this.peerConn.localDescription
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
    return {video: true, audio: true}
}