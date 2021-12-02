package services

import (
	"encoding/json"
	"fmt"
	"whub/hub_client"
	"whub/hub_client/controllers"
	"whub/hub_common/messages"
	"whub/hub_common/roles"
	"whub/hub_common/service"
)

const FileServiceID = "file"
const (
	FileServiceRouteGet     = "/get/:fileName"
	FileServiceRouteGetInfo = "/get/:fileName/info"
	FileServiceGetSection   = "/get/:fileName/:section"
	FileServiceRouteListAll = "/list"
	FileServiceStreamFile   = "/stream/:fileName"

	FileServicePath        = "fs"
	FileServiceSectionSize = 1024 * 10
)

type FileService struct {
	hub_client.IClientService
	fileController controllers.IFileController
}

func (s *FileService) Init(server roles.ICommonServer) (err error) {
	defer func() {
		s.Logger().Println("service has been initiated with err ", err)
	}()
	s.IClientService = hub_client.NewClientService(FileServiceID, "file server", service.ServiceAccessTypeBoth, service.ServiceExecutionSync, server)
	s.fileController, err = controllers.NewFileController(fmt.Sprintf("./%s", FileServicePath), FileServiceSectionSize)
	if err != nil {
		return err
	}
	return s.InitHandlers(service.NewRequestHandlerMapBuilder().
		Get(FileServiceRouteGet, s.Get).
		Get(FileServiceRouteListAll, s.List).Build())
}

func (s *FileService) Get(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	path := pathParams["fileName"]
	if path == "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "invalid path")
	}
	data, err := s.fileController.GetFile(path)
	if err != nil {
		return err
	}
	return s.ResolveByResponse(request, data)
}

func (s *FileService) List(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	stats, err := s.fileController.List(".")
	if err != nil {
		return err
	}
	marshalled, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return s.ResolveByResponse(request, marshalled)
}
