window.backEndAddress = () => { // TODO: of course, it would be awesome to handle errors here...
    var xmlHttp = new XMLHttpRequest();
    xmlHttp.open( "GET", `${window.location.protocol}//${window.location.host}/backend`, false ); // false for synchronous request
    xmlHttp.send( null );
    return xmlHttp.responseText;
}