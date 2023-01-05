# boos (aka pion-the-sky)

This project is an example of a simple golang media server implemented using [Pion webrtc](https://github.com/pion/webrtc). It records video and audio from a browser client and streams it back. 

# Supported Browsers

Tested so far on the following browsers:

- macOS
  - Firefox 108
  - Chrome 108
  - Safari 16

# Learnings

I encountered a few issues that required workarounds worth noting. You can read thru the code for more details: 

1. https://github.com/pion/webrtc/issues/716
2. https://stackoverflow.com/questions/47990094/failed-to-set-remote-video-description-send-parameters-on-native-ios

# Todo

1. The service merely stores the recorded video in memory for playback. Currently only the last stream will be recorded and played back. I need to restore 

# How to Run the Example...

```sh
go mod tidy
go run ./... all
```

1. Open a browser and go to `http://localhost:8083`.
2. Record a video at `http://localhost:8083/record`.
3. Hit the back button (or optionally disconnect and then hit the back button).
4. Play the videos you recorded.
