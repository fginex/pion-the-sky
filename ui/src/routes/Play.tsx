import React, { useEffect, useState } from 'react';

import { loadBackendData, getWebsocketAddress } from '../backend/backend'
import { Signaling, SignalSocketConnectStatus, SignalSocketError } from '../signaling/signaling'

const Play = () => {

    const [webSocketAddress, setWebsocketAddress] = useState<string | null>(null)

    const doConnect = () => {

        loadBackendData().then(response => {
            console.log("Loaded backend data", response.data)
            const wsAddress = getWebsocketAddress(response.data)
            if (wsAddress === null) {
                console.log(`Address ${response.data.address} does not to be a valid address`)
            } else {
                console.log("Loaded websocket address", wsAddress)
                setWebsocketAddress(wsAddress)
            }
        }).catch(e => console.log("Failed fetching backend data", e))

    }

    const doDisconnect = () => {
        setWebsocketAddress(null)
    }

    const doPlay = () => {

    }

    const doPrintCodecs = () => {
        
    }

    const doPrintSDS = () => {
        
    }

    const onSignalSocketConnected = () => {
        console.log("Signal socket: connected")
    }
    const onSignalSocketDisconnected = () => {
        console.log("Signal socket: disconnected")
    }
    const onSignalSocketServerError = (e: SignalSocketError) => {
        console.log(`Signal socket: server error, reason: ${e.reason}`)
    }
    const onSignalSocketConnectionProgress = (status: SignalSocketConnectStatus) => {
        console.log("Signal socket: connection progress", status.state)
    }
    const onSignalSocketRemoteSDP = (sdp: string) => {
        console.log("Signal socket: received data from signal server, streaming initiated, SDP", sdp)
    }

    return (
        <>
            <Signaling webSocketAddress={webSocketAddress}
                onConnected={onSignalSocketConnected}
                onConnectionProgress={onSignalSocketConnectionProgress}
                onDisconnected={onSignalSocketDisconnected}
                onRemoteSDP={onSignalSocketRemoteSDP}
                onServerError={onSignalSocketServerError} />

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