import axios, { AxiosResponse } from 'axios';

export interface BackEndData {
    address: string,
    iceServers: string[],
}

const loadBackendData = (): Promise<AxiosResponse<BackEndData, any>> => {
   return axios.get<BackEndData>("/backend")
}

export default loadBackendData;