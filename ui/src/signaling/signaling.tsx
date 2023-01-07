import { useEffect, useState } from 'react'

enum SignalOp {
	Undefined = "UNDEFINED",
	Record = "RECORD",
	Answer = "ANSWER",
	Play = "PLAY",
	Error = "ERROR",
}

interface SignalMessage {
    id: number
    op: SignalOp
    data: string
}

export enum ConnectState {
    AlreadyConnected,
    AlreadyConnecting,
	Connecting,
    Failed,
    InvalidWebSocketAddress,
}

export interface SignalSocketConnectStatus {
    state: ConnectState
    error?: SignalSocketError
}

export interface SignalSocketError {
    reason: string | Event
}

export interface SignalingSocketProps {
    webSocketAddress: string | null
    onConnected?: () => void
    onConnectionProgress?: (status: SignalSocketConnectStatus) => void
    onDisconnected?: () => void
    onError?: (error: SignalSocketError) => void
    onRemoteSDP: (sdp: string) => void
    onServerError?: (error: SignalSocketError) => void
}

export const Signaling = (props: SignalingSocketProps) => {

    const [signalSocket, setSignalSocket] = useState<WebSocket | null>(null)
    const [connecting, setConnecting] = useState(false)

    const maybeUpdateConnectionProgress = (state: ConnectState) => {
        if (props.onConnectionProgress !== undefined) {
            props.onConnectionProgress({state: state})
        }
    }

    useEffect(() => {
        if (props.webSocketAddress !== null) {
            try { new URL(props.webSocketAddress) } catch (e: any) {
                maybeUpdateConnectionProgress(ConnectState.InvalidWebSocketAddress)
                return
            }
            if (signalSocket !== null) {
                maybeUpdateConnectionProgress(ConnectState.AlreadyConnected)
                return
            }
            if (connecting) {
                maybeUpdateConnectionProgress(ConnectState.AlreadyConnecting)
                return
            }
            setConnecting(true)
            setSignalSocket(new WebSocket(props.webSocketAddress))
        } else {
            if (signalSocket !== null) {
                signalSocket.close()
            }
        }
    }, [props.webSocketAddress])

    useEffect(() => {
        if (signalSocket !== null) {

            maybeUpdateConnectionProgress(ConnectState.Connecting)
            
            signalSocket.onopen = (ev: Event): any => {
                setConnecting(false)
                if (props.onConnected !== undefined) {
                    props.onConnected()
                }
            }

            signalSocket.onerror = (ev: Event) => {
                if (props.onError !== undefined) {
                    props.onError({reason: ev})
                }
            }

            signalSocket.onclose = (ev: CloseEvent) => {
                setSignalSocket(null)
                // If the WebSocket cannot connect, the onerror on onclose will be called.
                // Let's set this.connecting to false here.
                setConnecting(false)
                if (props.onDisconnected !== undefined) {
                    props.onDisconnected()
                }
            }

            signalSocket.onmessage = (ev: MessageEvent<any>) => {
                let evt: SignalMessage = JSON.parse(ev.data)
                switch (evt.op) {
                    case SignalOp.Answer:
                        let encodedSDP = evt.data
                        let decodedSDP = Buffer.from(encodedSDP, "base64").toString()
                        props.onRemoteSDP(decodedSDP)
                        break
                    case SignalOp.Error:
                        if (props.onServerError !== undefined) {
                            props.onServerError({reason: evt.data})
                        }
                        break
                    default:
                        console.log(`Signal socket: unknown event received ${evt.op}`)
                }
            }

        } else {
            if (connecting) {
                maybeUpdateConnectionProgress(ConnectState.Failed)
                setConnecting(false)
            }
        }
    }, [signalSocket])



    return (<></>)
}
