package controllers
import (
	"strings"
	"wsdk/common/reflect"
)


// TODO bad, use common.ioc to do this!!!

const ControllerPrefix = "ctr: "
const ControllerInterface = "IController"

type ControllerContainer struct {
	controllers map[string]IController
}

func (c *ControllerContainer) Get(id string) IController {
	return c.controllers[id]
}

func (c *ControllerContainer) Set(id string, controller IController) {
	c.controllers[id] = controller
}

func (c *ControllerContainer) ScanAndInitialize(obj interface{}) error {
	fieldTags, err := reflect.GetFieldsAndTags(obj)
	if err != nil {
		return err
	}
	// tag starts with "ctr: " look for id
	// tag equals "ctr" first look by fieldName
	for k, v := range fieldTags {
		fieldType, err := reflect.GetFieldTypeByName(obj, k)
		if err != nil {
			return err
		}
		if fieldType
		if strings.HasPrefix(v, ControllerPrefix) &&  fieldType == "IController" {

		}
	}
}