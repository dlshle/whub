package ioc

import (
	"errors"
	"fmt"
	rawReflect "reflect"
	"strings"
	"sync"
	"wsdk/common/reflect"
)

const (
	TypeComponentPrefix = "$type-"
	AutoWireTagPrefix   = "$autowire"
	InjectTagPrefix     = "$inject:"
)

type Container struct {
	components map[string]interface{} // byTypePrefix
	tagPrefix  string                 // prefix: xxx
	lock       *sync.RWMutex
}

func NewContainer() *Container {
	return &Container{
		components: make(map[string]interface{}),
		lock:       new(sync.RWMutex),
	}
}

func (c *Container) withWrite(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *Container) GetById(id string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.components[id]
}

func (c *Container) GetByType(typeName string) interface{} {
	return c.GetById(c.assembleTypeId(typeName))
}

func (c *Container) registerComponent(id string, component interface{}) bool {
	notExist := c.GetById(id) == nil
	c.withWrite(func() {
		c.components[id] = component
	})
	return notExist
}

func (c *Container) assembleTypeId(typeName string) string {
	return fmt.Sprintf("%s-%s", TypeComponentPrefix, typeName)
}

func (c *Container) assembleTypedComponentId(component interface{}) string {
	return c.assembleTypeId(reflect.GetObjectType(component))
}

// register by type, will replace the last component registered under the same type
func (c *Container) AutoRegister(component interface{}) bool {
	return c.registerComponent(c.assembleTypedComponentId(component), component)
}

// register by fieldName
func (c *Container) AutoRegisterField(object interface{}, fieldName string) (bool, error) {
	value, err := reflect.GetValueByField(object, fieldName)
	if err != nil {
		return false, err
	}
	return c.RegisterComponent(fieldName, value.Interface()), nil
}

func (c *Container) RegisterComponent(id string, component interface{}) bool {
	return c.registerComponent(id, component)
}

func (c *Container) getComponentWithTagCheck(object interface{}, fieldName, prefix string) (interface{}, error) {
	tag, err := reflect.GetTagByField(object, fieldName)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(tag, prefix) {
		return nil, errors.New(fmt.Sprintf("prefix(%s) is not found at tag %s in field %s", prefix, tag, fieldName))
	}
	id := strings.TrimPrefix(tag, prefix)
	component := c.GetById(id)
	if component == nil {
		return nil, errors.New(fmt.Sprintf("assign failed for field %s as component with id %s does not exist", fieldName, id))
	}
	return component, nil
}

// $inject: xxx
func (c *Container) injectFieldById(object interface{}, fieldName string, tag string) error {
	if !strings.HasPrefix(tag, InjectTagPrefix) {
		return errors.New(fmt.Sprintf("inject prefix(%s) is not found at tag %s in field %s", InjectTagPrefix, tag, fieldName))
	}
	id := strings.TrimPrefix(tag, InjectTagPrefix)
	component := c.GetById(id)
	fmt.Println("component: ", component)
	if component == nil {
		return errors.New(fmt.Sprintf("injection failed for field %s as component with id %s does not exist", fieldName, id))
	}
	return reflect.SetValueOnField(object, fieldName, component)
}

func (c *Container) InjectField(object interface{}, fieldName string) error {
	tag, err := reflect.GetTagByField(object, fieldName)
	if err != nil {
		return err
	}
	return c.injectFieldById(object, fieldName, tag)
}

func (c *Container) InjectFields(object interface{}) error {
	ftMap, err := reflect.GetFieldsAndTags(object)
	if err != nil {
		return err
	}
	for k, v := range ftMap {
		if err = c.injectFieldById(object, k, v); err != nil {
			return err
		}
	}
	return nil
}

// $autowire
// first by type and then by name
func (c *Container) autowireFieldByField(object interface{}, field rawReflect.StructField) error {
	tag := string(field.Tag)
	if tag != AutoWireTagPrefix {
		return errors.New(fmt.Sprintf("incorrect tag(%s) for AutoWiring(%s)", tag, AutoWireTagPrefix))
	}
	typeName := field.Type.Name()
	if typeName == "" {
		// special case for struct { *AnotherStructName }
		typeName = field.Type.Elem().Name()
		if typeName == "" {
			typeName = field.Name
		}
	}
	component := c.GetByType(typeName)
	if component == nil {
		component = c.GetById(field.Name)
		if component == nil {
			return errors.New(fmt.Sprintf("autowiring failed for field %s as component could not be found by type(%s) or id(%s)",
				field.Name,
				c.assembleTypeId(field.Type.Name()),
				field.Name))
		}
	}
	return reflect.SetValueOnField(object, field.Name, component)
}

func (c *Container) AutoWireFieldByType(object interface{}, fieldName string) error {
	field, err := reflect.GetFieldByName(object, fieldName)
	if err != nil {
		return err
	}
	return c.autowireFieldByField(object, field)
}

func (c *Container) AutoWireFieldsByType(object interface{}) error {
	fields, err := reflect.GetFields(object)
	if err != nil {
		return err
	}
	for i := range fields {
		if err = c.autowireFieldByField(object, fields[i]); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) handleAutoInjectField(object interface{}, field rawReflect.StructField) error {
	tag := string(field.Tag)
	if strings.HasPrefix(tag, InjectTagPrefix) {
		return c.injectFieldById(object, field.Name, tag)
	} else if strings.HasPrefix(tag, AutoWireTagPrefix) {
		return c.autowireFieldByField(object, field)
	}
	return nil
}

func (c *Container) AutoInjectComponents(object interface{}) error {
	fields, err := reflect.GetFields(object)
	if err != nil {
		return err
	}
	for i := range fields {
		if err = c.handleAutoInjectField(object, fields[i]); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) UnregisterById(id string) bool {
	notExist := c.GetById(id) == nil
	if notExist {
		return false
	}
	c.withWrite(func() {
		delete(c.components, id)
	})
	return true
}

func (c *Container) Clear() {
	c.withWrite(func() {
		for k := range c.components {
			delete(c.components, k)
		}
	})
}
