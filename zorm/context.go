package zorm

import (
	"errors"
	"github.com/caixr9527/zorm/binding"
	zormLog "github.com/caixr9527/zorm/log"
	"github.com/caixr9527/zorm/render"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
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
	StatusCode            int
	Logger                *zormLog.Logger
	Keys                  map[string]any
	mu                    sync.RWMutex
	sameSite              http.SameSite
}

func (c *Context) SetSamSite(s http.SameSite) {
	c.sameSite = s
}

func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}
	c.Keys[key] = value
	c.mu.Unlock()
}

func (c *Context) Get(key string) (value any, ok bool) {
	c.mu.RLock()
	value, ok = c.Keys[key]
	c.mu.RUnlock()
	return
}

func (c *Context) SetBasicAuth(username, password string) {
	c.R.Header.Set("Authorization", "Basic "+BasicAuth(username, password))
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

func (c *Context) BindJson(obj any) error {
	jsonBinding := binding.JSON
	jsonBinding.DisallowUnknownFields = true
	jsonBinding.IsValidate = true
	return c.MustBindWith(obj, jsonBinding)
}

func (c *Context) BindXML(obj any) error {
	return c.MustBindWith(obj, binding.XML)
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
	err := r.Render(c.W, statusCode)
	c.StatusCode = statusCode
	return err
}

func (c *Context) MustBindWith(obj any, bind binding.Binding) error {
	if err := c.ShouldBindWith(obj, bind); err != nil {
		c.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

func (c *Context) ShouldBindWith(obj any, bind binding.Binding) error {
	return bind.Bind(c.R, obj)
}

func (c *Context) Fail(code int, msg string) {
	c.String(code, msg)
}

func (c *Context) HandlerWithError(statusCode int, obj any, err error) {
	if err != nil {
		code, data := c.engine.errorHandler(err)
		c.JSON(code, data)
		return
	}
	c.JSON(statusCode, obj)

}

func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(c.W, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		Path:     path,
		Domain:   domain,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: c.sameSite,
	})
}
