# Relay Center

## What does it do?
Relay center is a TCP server that managers and relays requests to service providers.

A service provider is a relay client that handles certain service logic(e.g. a micro-service 
or reads a file, etc) and returns the result back to the server.

Ideally, one client handles one service(maybe will handler multiple services in the future).

A service provider has to use an async connection(TCP/WebSocket/maybe UDP) to communicate with the server.

## TODO Tasks
* Better client design
  * No connection inside client
  * When needed, use ConnectionManager to get connections of a client
* Client manager w/ DB
* Proper service provider registration
* Common auth layer
* Proper async client auth
* Multiple connection per service provider
* Maybe multiple clients per service(for better availability)
* Separation of client access layer and service layer(service and service provider management)
  * Client request to access layer, access layer proxies the request to service layer, service will then handle the request
  * Service provider connects to the service layer directly to maintain continues connection
* Client SDK for other languages(Java, TypeScript, Rust)
* UDP connection
* WebRTC supports
* Distributed server
* If possible, one listener for all protocols