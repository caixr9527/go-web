package zorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/caixr9527/zorm/render"
	"github.com/go-playground/validator/v10"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
)

const defaultMaxMemory = 32 << 20

type Context struct {
	W                     http.ResponseWriter
	R                     *http.Request
	engine                *Engine
	queryParams           url.Values
	formParams            url.Values
	DisallowUnknownFields bool
	IsValidate            bool
}

func (c *Context) GetQuery(key string) string {
	c.initQueryParams()
	return c.queryParams.Get(key)
}

func (c *Context) GetDefaultQuery(key, defaultValue string) string {
	c.initQueryParams()
	values, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return values[0]
}

func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.initQueryParams()
	values, ok := c.queryParams[key]
	return values, ok
}

func (c *Context) initQueryParams() {
	if c.R != nil {
		c.queryParams = c.R.URL.Query()
	} else {
		c.queryParams = url.Values{}
	}
}

func (c *Context) initPostFormParams() {
	if c.R != nil {
		if err := c.R.ParseMultipartForm(defaultMaxMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formParams = c.R.PostForm
	} else {
		c.formParams = url.Values{}
	}
}

func (c *Context) GetPostFormArray(key string) ([]string, bool) {
	c.initPostFormParams()
	values, ok := c.formParams[key]
	return values, ok
}

func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initPostFormParams()
	return c.get(c.formParams, key)
}

func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}

func (c *Context) PostFormMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetPostFormMap(key)
	return
}

func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryParams()
	return c.get(c.queryParams, key)
}

func (c *Context) get(params map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	for k, value := range params {
		if index := strings.IndexByte(k, '['); index >= 1 && k[0:index] == key {
			if j := strings.IndexByte(k[index+1:], ']'); j >= 1 {
				exist = true
				dicts[k[index+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
}

func (c *Context) FormFile(name string) *multipart.FileHeader {
	file, header, err := c.R.FormFile(name)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	return header
}

func (c *Context) FormFiles(name string) ([]*multipart.FileHeader, error) {
	multipartForm, err := c.MultipartForm()
	return multipartForm.File[name], err
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.R.ParseMultipartForm(defaultMaxMemory)
	return c.R.MultipartForm, err
}

func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

func (c *Context) DealJson(obj any) error {
	body := c.R.Body
	if body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	// todo 两个同时开启需要同时支持 同时开启DisallowUnknownFields会失效
	if c.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if c.IsValidate {
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

type SliceValidationError []error

func (err SliceValidationError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]:%s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 0; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]:%s", i, err[0].Error())
				}
			}
		}
		return b.String()
	}
}

func validate(obj any) error {
	of := reflect.ValueOf(obj)
	switch of.Kind() {
	case reflect.Pointer:
		return validate(of.Elem().Interface())
	case reflect.Struct:
		return validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := of.Len()
		sliceValidationError := make(SliceValidationError, 0)
		for i := 0; i < count; i++ {
			if err := validateStruct(of.Index(i).Interface()); err != nil {
				sliceValidationError = append(sliceValidationError, err)
			}
		}
		return sliceValidationError
	}
	return nil
}

func validateStruct(obj any) error {
	return validator.New().Struct(obj)
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

func (c *Context) HTML(status int, html string) error {
	return c.Render(status, &render.HTML{Data: html, IsTemplate: false})
}

func (c *Context) HTMLTemplate(name string, data any, filenames ...string) error {

	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {

	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseGlob(pattern)
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

func (c *Context) Template(name string, data any) error {

	return c.Render(http.StatusOK, &render.HTML{
		Data:       data,
		IsTemplate: true,
		Template:   c.engine.HTMLRender.Template,
		Name:       name,
	})
}

func (c *Context) JSON(status int, data any) error {

	return c.Render(status, &render.JSON{Data: data})
}

func (c *Context) XML(status int, data any) error {
	return c.Render(status, &render.XML{Data: data})
}

func (c *Context) File(filename string) {
	http.ServeFile(c.W, c.R, filename)
}

func (c *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}

func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)

	c.R.URL.Path = filepath
	http.FileServer(fs).ServeHTTP(c.W, c.R)

}

func (c *Context) Redirect(status int, url string) error {
	return c.Render(status, &render.Redirect{Code: status, Request: c.R, Location: url})
}

func (c *Context) String(status int, format string, values ...any) error {
	return c.Render(status, &render.String{Format: format, Data: values})
}

func (c *Context) Render(statusCode int, r render.Render) error {
	err := r.Render(c.W)
	if statusCode != http.StatusOK {
		c.W.WriteHeader(statusCode)
	}
	return err
}
