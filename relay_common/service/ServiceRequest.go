package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"wsdk/common/async"
	"wsdk/relay_common/messages"
)

var requestPool sync.Pool

var ServiceRequestMessageHandlerTypes []int

func init() {
	initRequestPool()
	initServiceRequest()
	ServiceRequestMessageHandlerTypes = []int{
		messages.MessageTypeServiceRequest,
		messages.MessageTypeServiceGetRequest,
		messages.MessageTypeServicePostRequest,
		messages.MessageTypeServicePutRequest,
		messages.MessageTypeServicePatchRequest,
		messages.MessageTypeServiceDeleteRequest,
		messages.MessageTypeServiceOptionsRequest,
		messages.MessageTypeServiceHeadRequest,
	}
}

func initRequestPool() {
	requestPool = sync.Pool{New: func() interface{} {
		return &ServiceRequest{
			status:         ServiceRequestStatusQueued,
			barrier:        async.NewStatefulBarrier(),
			requestContext: make(map[string]interface{}),
		}
	}}
}

const (
	ServiceRequestStatusQueued     = 0
	ServiceRequestStatusProcessing = 1
	ServiceRequestStatusDead       = 2 // when health check failed
	ServiceRequestStatusFinished   = 3
	ServiceRequestStatusCancelled  = 4

	ServiceRequestContextUriPattern  = "uri_pattern"
	ServiceRequestContextPathParams  = "path_params"
	ServiceRequestContextQueryParams = "query_params"
)

var UnProcessableServiceRequestMap map[int]bool
var statusCodeStringMap map[int]string

func initServiceRequest() {
	statusCodeStringMap = make(map[int]string)
	statusCodeStringMap[ServiceRequestStatusQueued] = "queued"
	statusCodeStringMap[ServiceRequestStatusProcessing] = "processing"
	statusCodeStringMap[ServiceRequestStatusDead] = "dead"
	statusCodeStringMap[ServiceRequestStatusFinished] = "finished"
	statusCodeStringMap[ServiceRequestStatusCancelled] = "cancelled"

	UnProcessableServiceRequestMap = make(map[int]bool)
	UnProcessableServiceRequestMap[ServiceRequestStatusDead] = true
	UnProcessableServiceRequestMap[ServiceRequestStatusCancelled] = true
}

type ServiceRequest struct {
	barrier *async.StatefulBarrier
	status  int
	messages.IMessage
	requestContext map[string]interface{}
}

func NewServiceRequest(m messages.IMessage) IServiceRequest {
	request := requestPool.Get().(*ServiceRequest)
	request.requestContext = make(map[string]interface{})
	request.barrier = async.NewStatefulBarrier()
	request.status = ServiceRequestStatusQueued
	request.IMessage = m
	return request
}

type IServiceRequest interface {
	messages.IMessage
	Message() messages.IMessage
	Status() int
	Kill() error
	Cancel() error
	IsDead() bool
	IsCancelled() bool
	IsFinished() bool
	Resolve(messages.IMessage) error
	Wait() error // wait for the state to transit to final (dead/finished/cancelled)
	Response() messages.IMessage
	TransitStatus(int)
	GetContext(key string) interface{}
	SetContext(key string, value interface{})
	Free()
}

func (t *ServiceRequest) setStatus(status int) {
	if t.Status() == ServiceRequestStatusFinished {
		// can not set status of a finished service message_dispatcher
		return
	}
	t.status = status
}

func (t *ServiceRequest) Status() int {
	return t.status
}

func (t *ServiceRequest) Kill() error {
	if t.Status() > 1 {
		return errors.New("unable to kill a " + statusCodeStringMap[t.Status()] + " ServiceRequest")
	}
	t.status = ServiceRequestStatusDead
	t.barrier.OpenWith(nil)
	return nil
}

func (t *ServiceRequest) Cancel() error {
	if t.Status() > 1 {
		return errors.New("unable to cancel a " + statusCodeStringMap[t.Status()] + " ServiceRequest")
	}
	t.status = ServiceRequestStatusCancelled
	t.barrier.OpenWith(nil)
	return nil
}

func (t *ServiceRequest) Resolve(m messages.IMessage) error {
	if t.Status() != ServiceRequestStatusProcessing {
		return errors.New("can not Resolve a non-processing ServiceRequest")
	}
	t.status = ServiceRequestStatusFinished
	t.barrier.OpenWith(m)
	return nil
}

func (t *ServiceRequest) IsDead() bool {
	return t.Status() == ServiceRequestStatusDead
}

func (t *ServiceRequest) IsCancelled() bool {
	return t.Status() == ServiceRequestStatusCancelled
}

func (t *ServiceRequest) IsFinished() bool {
	return t.Status() == ServiceRequestStatusFinished
}

func (t *ServiceRequest) Wait() error {
	if t.Status() != ServiceRequestStatusProcessing {
		return errors.New("can not wait for a non-processing ServiceRequest")
	}
	t.barrier.Wait()
	return nil
}

func (t *ServiceRequest) Response() messages.IMessage {
	return t.barrier.Get().(messages.IMessage)
}

func (t *ServiceRequest) TransitStatus(status int) {
	t.setStatus(status)
}

func (t *ServiceRequest) SetContext(key string, value interface{}) {
	t.requestContext[key] = value
}

func (t *ServiceRequest) GetContext(key string) interface{} {
	return t.requestContext[key]
}

func (t *ServiceRequest) Message() messages.IMessage {
	return t.IMessage
}

func (t *ServiceRequest) String() string {
	ctxBytes, err := json.Marshal(t.requestContext)
	if err != nil {
		ctxBytes = ([]byte)("{}")
	}
	return fmt.Sprintf("{\"id\": \"%s\", \"messsage\": %s, \"context\": %s}", t.Id(), t.Message().String(), (string)(ctxBytes))
}

func (t *ServiceRequest) Free() {
	t.requestContext = nil
	t.barrier = nil
	requestPool.Put(t)
}
