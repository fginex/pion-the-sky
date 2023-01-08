import axios, { AxiosResponse } from 'axios';

export interface BackendData {
    address: string,
    iceServers: string[],
}

export interface BackendConfig {
    config: BackendData
    webSocketAddress: string
}

const loadBackendData = (): Promise<AxiosResponse<BackendData, any>> => {
   return axios.get<BackendData>("/backend")
}

export const getBackendConfig = (): Promise<BackendConfig> => {
    return new Promise<BackendConfig>((resolve, reject) => {
        loadBackendData()
            .then(backendData => {
                const address = new URL(backendData.data.address)
                if (address.protocol === "https:") {
                    resolve({config: backendData.data, webSocketAddress: `wss://${address.host}/ws`})
                } else if (address.protocol === "http:") {
                    resolve({config: backendData.data, webSocketAddress: `ws://${address.host}/ws`})
                } else {
                    reject(`unexpected protocol ${address.protocol}, expected https: or http:`)
                }
            })
            .catch(e => reject(e))
    })
}
