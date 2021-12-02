package service

import "whub/hub_common/messages"

type IRequestHandlerMap interface {
	Add(requestType int, uri string, handler RequestHandler) IRequestHandlerMap
	Get(uri string, handler RequestHandler) IRequestHandlerMap
	Post(uri string, handler RequestHandler) IRequestHandlerMap
	Put(uri string, handler RequestHandler) IRequestHandlerMap
	Patch(uri string, handler RequestHandler) IRequestHandlerMap
	Delete(uri string, handler RequestHandler) IRequestHandlerMap
	Head(uri string, handler RequestHandler) IRequestHandlerMap
	Options(uri string, handler RequestHandler) IRequestHandlerMap
	Build() map[int]map[string]RequestHandler
}

type RequestHandlerMap struct {
	handlersMap map[int]map[string]RequestHandler
}

func NewRequestHandlerMapBuilder() *RequestHandlerMap {
	return &RequestHandlerMap{
		handlersMap: make(map[int]map[string]RequestHandler),
	}
}

func (b *RequestHandlerMap) Add(requestType int, uri string, handler RequestHandler) IRequestHandlerMap {
	if b.handlersMap[requestType] == nil {
		b.handlersMap[requestType] = make(map[string]RequestHandler)
	}
	b.handlersMap[requestType][uri] = handler
	return b
}

func (b *RequestHandlerMap) Get(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServiceGetRequest, uri, handler)
}

func (b *RequestHandlerMap) Post(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServicePostRequest, uri, handler)
}

func (b *RequestHandlerMap) Put(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServicePutRequest, uri, handler)
}

func (b *RequestHandlerMap) Patch(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServicePatchRequest, uri, handler)
}

func (b *RequestHandlerMap) Delete(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServiceDeleteRequest, uri, handler)
}

func (b *RequestHandlerMap) Head(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServiceHeadRequest, uri, handler)
}

func (b *RequestHandlerMap) Options(uri string, handler RequestHandler) IRequestHandlerMap {
	return b.Add(messages.MessageTypeServiceOptionsRequest, uri, handler)
}

func (b *RequestHandlerMap) Build() map[int]map[string]RequestHandler {
	return b.handlersMap
}
