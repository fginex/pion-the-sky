import { BackendConfig } from '../backend/backend'
import { Buffer } from 'buffer'

export enum SignalOp {
	Undefined = "UNDEFINED",
	Record = "RECORD",
	Answer = "ANSWER",
	Play = "PLAY",
	Error = "ERROR",
}

export interface SignalMessage {
    id?: number
    op: SignalOp
    data: string
}

export interface SignalingError {
    reason: string
    cause?: any
}

export class ConnectedSignaling {

    private socket: WebSocket

    private inProgress: Promise<string> | null = null
    private inProgressResolve: (value: string | PromiseLike<string>) => void = (_: string | PromiseLike<string>) => {}
    private inProgressReject: (reason?: any) => void = (_?: any) => {}

    private close: Promise<Event> | null = null
    private closeResolve: (value: Event | PromiseLike<Event>) => void = (_: Event | PromiseLike<Event>) => {}

    private closed = false

    constructor(ws: WebSocket) {
        this.socket = ws
        this.socket.onclose = this.onSocketClose.bind(this)
        this.socket.onerror = this.onSocketError.bind(this)
        this.socket.onmessage = this.onSocketMessage.bind(this)
    }

    play(sdp: string) {
        // defences
        let state = this.validState()
        if (state !== null) {
            return state
        }
        // do work
        return this.signalMessage(SignalOp.Play, sdp)
    }

    record(sdp: string) {
        // defences
        let state = this.validState()
        if (state !== null) {
            return state
        }
        // do work
        return this.signalMessage(SignalOp.Record, sdp)
    }

    private validState() {
        if (this.closed) {
            return new Promise<RTCSessionDescription>((_, reject) => {
                const e: SignalingError = {
                    reason: "already closed",
                }
                reject(e)
            })
        }
        if (this.inProgress !== null) {
            return new Promise<RTCSessionDescription>((_, reject) => {
                const e: SignalingError = {
                    reason: "already servicing",
                }
                reject(e)
            })
        }
        return null
    }

    private signalMessage(op: SignalOp, sdp: string) {
        this.inProgress = new Promise<string>((resolve, reject) => {
            this.inProgressResolve = resolve.bind(this)
            this.inProgressReject = reject.bind(this)
        })
        const message: SignalMessage = {
            op: op,
            data: sdp,
        }
        this.socket.send(JSON.stringify(message))
        return this.inProgress
    }

    disconnect() {
        if (this.closed) {
            return new Promise<Event>((_, reject) => {
                const e: SignalingError = {
                    reason: "already closed",
                }
                reject(e)
            })
        }
        // do work
        this.close = new Promise<Event>((resolve, _) => {
            this.closeResolve = resolve
            this.socket.close()
        })
        return this.close
    }

    // socket events:
    private onSocketClose(ev: Event) {
        this.closed = true
        if (this.inProgress !== null) {
            this.reject("signaling disconnected", ev)
        }
        if (this.close !== null) {
            this.closeResolve(ev)
        }
    }

    // socket events:
    private onSocketError(ev: Event) {
        if (this.inProgress !== null) {
            this.reject("websocket error", ev)
        }
    }

    private onSocketMessage(e: MessageEvent<any>) {
        if (this.inProgress !== null) {
            const evt: SignalMessage = JSON.parse(e.data)
            switch (evt.op) {
                case SignalOp.Answer:
                    let encodedSDP = evt.data
                    try {
                        let decodedSDP = Buffer.from(encodedSDP, "base64").toString()
                        this.inProgressResolve(decodedSDP)
                    } catch (err) {
                        this.reject("failed decoding remote SDP", e)
                    }
                    break
                case SignalOp.Error:
                    this.reject("server error", evt.op)
                    break
                default:
                    this.reject("unexpected operation", evt.op)
                    break
            }
        }
    }

    private reject(reason: string, cause?: any) {
        let e: SignalingError = {
            reason: reason,
            cause: cause,
        }
        this.inProgressReject(e)
    }

}

export const signaling = (backendConfig: BackendConfig): Promise<ConnectedSignaling> => {
    return new Promise<ConnectedSignaling>((resolve, reject) => {
        try {
            let socket = new WebSocket(backendConfig.webSocketAddress)
            socket.onopen = (ev: Event): any => {
                resolve(new ConnectedSignaling(socket))
            }
            socket.onerror = (e => reject(e))
        } catch (e) { reject(e) }
    })
}