import React, { useEffect, useState } from 'react';

import loadBackendData, { BackEndData } from '../backend/backend'
import { getWebsocketAddress } from '../backend/websocket'

const Play = () => {

    const [backEndData, setBackEndData] = useState<BackEndData | null>(null)
    const [webSocketAddress, setWebsocketAddress] = useState<string | null>(null)

    const doConnect = () => {

    }

    const doDisconnect = () => {

    }

    const doPlay = () => {

    }

    const doPrintCodecs = () => {
        
    }

    const doPrintSDS = () => {
        
    }

    useEffect(() => {
        loadBackendData().then(response => {
            if (backEndData === null) {
                console.log("Loaded backend data", response.data)
                setBackEndData(response.data)
                const wsAddress = getWebsocketAddress(response.data)
                if (wsAddress === null) {
                    console.log(`Address ${response.data.address} does not to be a valid address`)
                } else {
                    console.log("Loaded websocket address", wsAddress)
                    setWebsocketAddress(wsAddress)
                }
            }
        }).catch(e => console.log("Failed fetching backend data", e))
    }, [backEndData])

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