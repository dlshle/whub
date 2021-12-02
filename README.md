## WHub

## Project Renaming
The project was named WSDK(WebSocket relay SDK) as it was specifically designed for message relaying using WebSocket. Later, as the project grows, it does more than relaying messages. Therefore, the project is renamed to WHub(WebSocket Hub). 

The project consists of three parts: Common, Server,  Client.

* Common contains common components shared by both Server and Client.
* Server consists of only server side logic(e.g. service management/middleware management/etc...)
* Client consists of only client side logic(e.g. server connection management/service framework/etc...)

Consideration of supporting TCP/UDP/WebRTC?
The project now supports TCP poorly(still dealing with sticky packet and packet separation issues). The reason WebSocket was considered as the first choice is because it deals with such issues internally so application does not have to deal with it.

## What does it do?
WebSocket Hub is a WebSocket server that manages and proxies serialized messages to service providers.

A service provider is a WebSocket client that handles certain service logic(e.g. a micro-service 
or read static files, etc) and responds the processed result as a WebSocket message back to the server.

Ideally, one client handles one service(maybe will handler multiple services in the future).

A service provider has to use an async connection(TCP/WebSocket/maybe UDP) to communicate with the server.

A service can also have multiple provider connections for better availability. Once a new provider registers a service, 
a new connection channel for the service is established.

## Structures of Hub Server and Hub Client
Hub Server consists of three parts: Server, Message Dispatcher, Modules.
* Server is where incoming connections are handled. By default, WebSocket Server is used for Hub Server.
* Message Dispatcher is where requests are handled.
* Module is where business logic is handled. ModuleManager manages the lifecycle of modules in Hub Server. All modules are registered when Server starts.

Hub Client is a little bit different from Hub Server where there are 3 main parts: Connection Pool, Message Dispatcher, Services.
* Connection Pool maintains a pool of server connections. Each time client tries to initiate a request, it will pick an available connection from connection pool(usually based on RoundRobin policy).
* Message Dispatcher is where requests are handled.
* Services are provided by client where other clients(not necessary Hub Client) can send request to via Hub Server.

For common component of both Hub Server and Hub Client, Message Dispatcher, it dispatches messages based on message type. Users can register different Message Handlers to Message Dispatcher at any phase of Hub Server or Hub Client.

## Hub Server Modules
In Hub Server, there are 3 important modules:
* ConnectionManagerModule manages all the socket connections(particularly, websocket connection).
* ClientManagerModule manages all the authorized clients.
* ServiceManagerModule manages all the services registered to the server.

Module details TBD.


## How does a message travel from client to server?
For an HTTP request, a typical scenario for an HTTP request would be:
Client initiates a TCP connection with server, server accepts the connection and receives the HTTP request from the client. Server initiates a new go routine to handle the HTTP request. 
In the request handler goroutine, the request will be transformed into a request message filled with extra meta-data in its header. The message is then sent to the Message Dispatcher. This is where the request handler goroutine finishes.
The Message Dispatcher dispatches the message in a dedicated goroutine, which is managed by the goroutine pool. In this goroutine, Message Dispatcher decides which message handler is used depending on the type of message. Normally, the message handler used is "Service Message Handler".
"Service Message Handler" first checks the path of the message(which is usually equivelent to the url path of the HTTP request), and try to find the corresponding "Service Request Handler" for it. If no "Service Request Handler" is found, "Service Message Handler" will respond NOT_FOUND_ERR to the client.
When the "Service Request Handler" is found, the "Service Request Handler" will handle the service message and respond the corresponding result.

For a serialized websocket message, a typical round trip from client to server would be:
WHub Client serialize the message using the "WProtocol" which is a message serialization protocol based on Flatbuffers. Client then picks a connection from server connection pool(usually based on RoundRobin policy) and send the message through the connection.
WHub Server receives the websocket message from client connection read loop goroutine. The websocket message is then deserialized using Flatbuffer, and then the deserialized message is sent to the Message Dispatcher, and Message Dispatcher will dispatch the message in a different gorountine. Client connection read loop then tries to read the next message until read error or connection close.
Message Dispatcher decides which message handler is used depending on the type of message. As usual, the handler used is "Service Message Handler".
"Service Message Handler" first checks the path of the message, and try to find the corresponding "Service Request Handler" for it. If no "Service Request Handler" is found, "Service Message Handler" will respond NOT_FOUND_ERR to the client.
When the "Service Request Handler" is found, the "Service Request Handler" will handle the service message and return the corresponding response to the "Service Message Handler". "Service Message Handler" will then send the serialized response back through the websocket connection.
Clinet receives the websocket message from server. Client the deserialize the message and notify the request initiator. Here the full round trip from client to server is finished.




## TODO Tasks
* Message header sanitizing
  * Option 1: discard all block-list header keys in a middleware
  * Option 2: middlewares to keep desired header keys
  * Option 3: headers in request are kept as request-headers, Hub Server keeps desired headers as response header.
    * Do we need a set of middlewares that run after request is handled?
* Better client design
  * No connection inside client
      * How? Make a sidecar for each client to handle connections?
  * ~~When needed, use ConnectionManager to get connections of a client~~
* ~~Client manager w/ DB~~
* Proper service provider registration
* ~~Common auth layer~~
* ~~Proper async client auth~~
* ~~Multiple connection per service provider~~
* ~~Maybe multiple clients per service(for better availability)~~
* Separation of client access layer and service layer(service and service provider management)
  * Client request to access layer, access layer proxies the request to service layer, service will then handle the request
  * Service provider connects to the service layer directly to maintain continues connection
  * What's the benefits of this compared to sidecars?
* Client SDK for other languages(Java, ~~TypeScript~~, Rust)
* Supporting HTTP Client connection(HTTP health check, ~~HTTP based RPC via Flatbuffers~~)
* UDP connection
* WebRTC supports
* Distributed server
  * Use ETCD for common state sharing(services, connections?, etc)
* If possible, one listener for all protocols
  * Rewriting net.Listener?