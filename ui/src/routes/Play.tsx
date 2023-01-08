import React, { useEffect, useState } from 'react';

import { getBackendConfig, BackendConfig } from '../backend/backend'
import { ConnectedPeerClient, peerConnectionWithMedia } from '../media/media2';
import { readCodecs } from '../media/sdp';
import { signaling, ConnectedSignaling } from '../signaling/signaling'

const Play = () => {

    const [backendConfig, setBackendConfig] = useState<BackendConfig | null>(null)
    const [connectedSignaling, setConnectedSignaling] = useState<ConnectedSignaling | null>(null)
    const [connectedPeerClient, setConnectedPeerClient] = useState<ConnectedPeerClient | null>(null)


    // const [mediaClient, setMediaClient] = useState<MediaClient | null>(null)
    // const [localSDP, setLocalSDP] = useState<string>("")

    const doConnect = () => getBackendConfig()
        .then(config => signaling(config)
            .then(result => {
                console.log("Signaling connected")
                setBackendConfig(config)
                setConnectedSignaling(result)
            })
            .catch(e => console.log("Failed connecting signaling", e))
        )
        .catch(e => console.log("Failed fetching backend config", e))

    const doDisconnect = () => {
        if (connectedSignaling !== null) {
            connectedSignaling.disconnect()
                .then(() => console.log("Signaling disconnected"))
                .catch(e => console.log("Signaling disconnect error", e))
                .finally(() => setConnectedSignaling(null))
        } else {
            console.log("Signaling not connected")
        }

        if (connectedPeerClient !== null) {
            connectedPeerClient.disconnectMedia()
                .then(() => console.log("Media disconnected"))
                .catch(e => console.log("Media disconnect error", e))
                .finally(() => setConnectedPeerClient(null))
        } else {
            console.log("Media not connected")
        }
    }

    const doPlay = () => {
        if (backendConfig !== null) {
            if (connectedSignaling !== null) {
                peerConnectionWithMedia({iceServers: backendConfig.config.iceServers})
                    .then(result => result.connectMedia()
                        .then(() => {
                            console.log("Peer client is now connected", result)
                            const ld = result.localDescription()
                            if (ld !== null) {
                                connectedSignaling.play(ld.sdp)
                                    .then(() => {
                                        console.log("Playing...")
                                        setConnectedPeerClient(result)
                                    })
                                    .catch(err => {
                                        console.log("Could not play media, reason:", err)
                                        result.disconnectMedia().finally(() => setConnectedPeerClient(null))
                                    })
                            } else {
                                console.log("Could not play media, reason:", "no local description on connected peer client")
                            }
                        })
                        .catch(err => console.log("Failed resolving user media", err)))
            } else {
                console.log("Signaling not connected")
            }
        } else {
            console.log("Backend not configured")
        }
    }

    const doPrintCodecs = () => {
        peerConnectionWithMedia({iceServers: []})
            .then(result => result.connectMedia()
                .then(() => {
                    console.log("Peer client is now connected and media resolved", result)
                    const mayLocalSessionDescription = result.localDescription()
                    if (mayLocalSessionDescription !== null) {
                        console.log("Local session description", mayLocalSessionDescription)
                        readCodecs(mayLocalSessionDescription)
                            .then(codecs => console.log(codecs))
                            .catch(err => console.log("Failed reading codec data from session description", err))
                    }
                    result.disconnectMedia()
                })
                .catch(err => console.log("Failed resolving user media", err))
            )
            .catch(err => console.log("Failed connecting peer client", err))
    }

    const doPrintSDS = () => {
        
    }

    return (
        <>
            <h2>Pion WebRTC - Record a sample</h2>
            <br />
            <button id="connectBtn" onClick={doConnect}>Connect</button>
            <button id="disconnectBtn" onClick={doDisconnect}>Disconnect</button>
            <pre></pre>
            <button id="playBtn" onClick={doPlay}>Play Stream</button>
            <button id="codecsBtn" onClick={doPrintCodecs}>Available Codecs</button>
            <button id="sdsBtn" onClick={doPrintSDS}>Session Desc</button>
            <br /><br />
            Video (Streaming playback)<br />
            <video id="remoteVideo" width="640" height="480" autoPlay controls></video> <br />
            <audio id="remoteAudio" autoPlay></audio> <br />
            <br /><br />___<br />
            <div id="logs"></div>
        </>
    )
}

export default Play
