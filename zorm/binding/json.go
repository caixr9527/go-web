package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

type jsonBinding struct {
	DisallowUnknownFields bool
	IsValidate            bool
}

func (b jsonBinding) Bind(r *http.Request, obj any) error {
	body := r.Body
	if body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	// todo 两个同时开启需要同时支持 同时开启DisallowUnknownFields会失效
	if b.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if b.IsValidate {
		err := validateParam(obj, decoder)
		if err != nil {
			return err
		}
	} else {
		err := decoder.Decode(obj)
		if err != nil {
			return err
		}

	}
	return validate(obj)
}

func (jsonBinding) Name() string {
	return "json"
}

func validateParam(obj any, decoder *json.Decoder) error {
	valueOf := reflect.ValueOf(obj)
	if valueOf.Kind() != reflect.Pointer {
		return errors.New("no ptr type")
	}
	elem := valueOf.Elem().Interface()
	of := reflect.ValueOf(elem)
	switch of.Kind() {
	case reflect.Struct:
		return checkParam(obj, decoder, of)
	case reflect.Slice, reflect.Array:
		elem := of.Type().Elem()
		if elem.Kind() == reflect.Struct {
			return checkParamSlice(elem, obj, decoder)
		}
		// todo 指针类型支持
	default:
		_ = decoder.Decode(obj)
	}
	return nil
}

func checkParam(obj any, decoder *json.Decoder, of reflect.Value) error {
	mapValue := make(map[string]interface{})
	_ = decoder.Decode(&mapValue)
	for i := 0; i < of.NumField(); i++ {
		field := of.Type().Field(i)
		name := field.Name
		jsonName := field.Tag.Get("json")
		if jsonName != "" {
			name = jsonName
		}
		required := field.Tag.Get("required")
		value := mapValue[name]
		if value == nil && required == "true" {
			return errors.New(fmt.Sprintf("field [%s] is not exist, because [%s] is required", jsonName, jsonName))
		}
	}

	marshal, _ := json.Marshal(mapValue)
	_ = json.Unmarshal(marshal, obj)
	return nil
}

func checkParamSlice(of reflect.Type, obj any, decoder *json.Decoder) error {
	mapValue := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapValue)
	for i := 0; i < of.NumField(); i++ {
		field := of.Field(i)
		name := field.Name
		jsonName := field.Tag.Get("json")
		if jsonName != "" {
			name = jsonName
		}
		required := field.Tag.Get("required")
		for _, v := range mapValue {
			value := v[name]
			if value == nil && required == "true" {
				return errors.New(fmt.Sprintf("field [%s] is not exist, because [%s] is required", jsonName, jsonName))
			}
		}
	}

	marshal, _ := json.Marshal(mapValue)
	_ = json.Unmarshal(marshal, obj)
	return nil
}
