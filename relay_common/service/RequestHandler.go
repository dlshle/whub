package service

type RequestHandler func(request *ServiceRequest, pathParams map[string]string, queryParams map[string]string) error
