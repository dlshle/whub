package services

import (
	"encoding/json"
	"fmt"
	"wsdk/relay_client"
	"wsdk/relay_client/controllers"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
)

const FileServiceID = "file"
const (
	FileServiceRouteGet     = "/get/:fileName"
	FileServiceRouteGetInfo = "/get/:fileName/info"
	FileServiceGetSection   = "/get/:fileName/:section"
	FileServiceRouteListAll = "/files"
	FileServiceStreamFile   = "/stream/:fileName"

	FileServicePath        = "fs"
	FileServiceSectionSize = 1024 * 10
)

type FileService struct {
	relay_client.IClientService
	fileController controllers.IFileController
}

func (s *FileService) Init(server roles.ICommonServer) (err error) {
	defer func() {
		s.Logger().Println("service has been initiated with err ", err)
	}()
	currPath, err := controllers.GetCurrentPath()
	if err != nil {
		return err
	}
	s.fileController, err = controllers.NewFileController(fmt.Sprintf("%s/%s", currPath, FileServicePath), FileServiceSectionSize)
	if err != nil {
		return err
	}
	s.IClientService = relay_client.NewClientService(FileServiceID, "file server", service.ServiceAccessTypeBoth, service.ServiceExecutionSync, server)
	err = s.RegisterRoute(FileServiceRouteGet, s.Get)
	if err != nil {
		return
	}
	return s.RegisterRoute(FileServiceRouteListAll, s.List)
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
