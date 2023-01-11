import {
    Link,
  } from "react-router-dom";

export default function Root() {
    return (
        <div>
            <h2>Pion WebRTC - Record and Playback as Stream Example</h2>
            <br />
            <Link to="/record/">Record your audio/video on the server</Link>
            <br />
            <Link to="/play/">Stream back your recordings</Link>
            <br />
        </div>
    )
}
