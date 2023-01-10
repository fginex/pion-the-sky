import { useRef, useState } from 'react';

import { getBackendConfig, BackendConfig } from '../backend/backend'
import { ConnectedPeerClient, peerConnectionWithMedia } from '../media/media2';
import { readCodecs } from '../media/sdp';
import { signaling, ConnectedSignaling } from '../signaling/signaling'

const Record = () => {

    const [backendConfig, setBackendConfig] = useState<BackendConfig | null>(null)
    const [connectedSignaling, setConnectedSignaling] = useState<ConnectedSignaling | null>(null)
    const [connectedPeerClient, setConnectedPeerClient] = useState<ConnectedPeerClient | null>(null)
    const [recordingInProgress, setRecordingInProgress] = useState(false)

    const previewVideoRef = useRef<HTMLVideoElement | null>(null)

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

    const doRecord = () => {
        if (backendConfig !== null) {
            if (connectedSignaling !== null) {
                peerConnectionWithMedia({iceServers: backendConfig.config.iceServers})
                    .then(result => result.connectMedia()
                        .then(stream => result.withCandidate().then(() => {
                            
                            const ld = result.localDescription()
                            if (previewVideoRef !== null) {
                                if (previewVideoRef.current !== null) {
                                    previewVideoRef.current.srcObject = stream
                                }
                            }

                            setTimeout(() => {
                                if (ld !== null) {
                                    connectedSignaling.record(ld)
                                        .then(remoteDP => {
                                            console.log("Recording...", remoteDP)
                                            result.setRemoteDescription(remoteDP)
                                            setRecordingInProgress(true)
                                            setConnectedPeerClient(result)
                                        })
                                        .catch(err => {
                                            console.log("Could not record media, reason:", err)
                                            result.disconnectMedia().finally(() => setConnectedPeerClient(null))
                                        })
                                } else {
                                    console.log("Candidate received but no local session description")
                                }
                            }, 5000)

                        })
                        .catch(err => console.log("Error on candidate wait", err)))
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
            <div><button id="codecsBtn" onClick={doPrintCodecs}>Available Codecs</button></div>
            { connectedSignaling === null
                ? <div><button id="connectBtn" onClick={doConnect}>Connect</button></div>
                : <div>
                    <button id="disconnectBtn" onClick={doDisconnect}>Disconnect</button>
                    <button id="playBtn" onClick={doRecord}>Record the Stream</button>
                    { recordingInProgress ? <span>Recording in progress</span> : <></> }
                    {
                        connectedPeerClient === null
                            ? <></>
                            : <div><button id="sdsBtn" onClick={doPrintSDS}>Session Desc</button></div>
                    }
                  </div> }
            
            <div>
                <h3>Video (Recording preview)</h3>
                <video ref={previewVideoRef} width="640" height="480" autoPlay muted></video> <br />
            </div>
            
        </>
    )
}

export default Record
