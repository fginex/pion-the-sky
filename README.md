# boos (aka pion-the-sky)

This project is an example of a simple golang media server implemented using [Pion webrtc](https://github.com/pion/webrtc). It records videos from a browser client and streams it back concatenated. 

# Goal

To demonstrate how pion can be used as a bolt-on intermediary to a proprietary media server for interacting with webrtc browser clients by showing that a/v packets can be manipulated at the rtp level and even lower with transcoding. Thus allowing an existing media service to be adapted such that it serves web clients on both desktop and mobile. 

# Components

The application consists of a service that listens on multiple endpoints:

1. Http requests and websocket connections are accepted on the default port 8082 or the port specified via the `-port=` command-line arg. This is the _Signal Service_. It is used to instantiate a compatible webrtc peer client on the server-side which will be used to either record or playback video depending upon the endpoint path (`/record` and `/play` respectively). When no path is specified (ie `http://localhost:8082`) the index.html will be served allowing you to select either record or play.

2. WebRTC Peer Connection Ports. The service will create a server-side peer client used to serve audio and video to the browser client.

# Codecs

Both **H264** and **VP8** video are supported, however the service is currently fixed to only use **Opus** as the audio codec. The video codec can be specified at startup via the `-vcodec=[vp8|h264]` command-line arg. The default is h264.  

# Supported Browsers

In progress... I have tested so far on the following browsers:
* macOS: (Chrome, Safari) 


# Learnings

I encountered a few issues that required workarounds worth noting. You can read thru the code for more details: 
1. https://github.com/pion/webrtc/issues/716
2. https://stackoverflow.com/questions/47990094/failed-to-set-remote-video-description-send-parameters-on-native-ios


# Todo

1. The service merely stores the recorded video in memory for playback. It uses pion's rtpdump.Writer to do so. There are lots of ways it could have been done. I just wanted to test out this particular method. 
2. Video appears to be recorded just fine but audio is a bit problematic.
  - Firefox 107 audio fails, most likely because the opus codec needs to be configured via negotiation.
  - Works fine in Safari and Chrome on macOS.

# How to Run the Example...

```sh
go mod tidy
go run ./... all
```

1. Open a browser and go to `http://localhost:8083`.
2. Record some videos. You can disconnect and reconnet to start and store a new video without refreshing the page.
3. Hit the back button (or optionally disconnect and then hit the back button).
4. Play the videos you recorded.
