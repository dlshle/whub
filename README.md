# Relay Center

## Project Renaming
The project was named WSDK(WebSocket relay SDK) as it was specifically designed for message relaying using WebSocket. Later, as the project grows, it does more than relaying messages. Therefore, the project is renamed to WHub(WebSocket Hub). 

The project consists of three parts: Common, Server,  Client.

* Common contains common components shared by both Server and Client.
* Server consists of only server side logic(e.g. service management/middleware management/etc...)
* Client consists of only client side logic(e.g. server connection management/service framework/etc...)

Consideration of supporting TCP/UDP/WebRTC?
The project now supports TCP poorly(still dealing with sticky packet and packet seperation issues). The reason WebSocket was considered as the first choice is because it deals with such issues internally so application does not have to deal with it.

## What does it do?
WebSocket Hub is a WebSocket server that manages and proxies serialized messages to service providers.

A service provider is a WebSocket client that handles certain service logic(e.g. a micro-service 
or reads a file, etc) and respondes the processed result as a WebSocket message back to the server.

Ideally, one client handles one service(maybe will handler multiple services in the future).

A service provider has to use an async connection(TCP/WebSocket/maybe UDP) to communicate with the server.

A service can also have multiple provider connections for better availability. Once a new provider registers a service, 
a new connection channel for the service is established.

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
