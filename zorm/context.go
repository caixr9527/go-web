package zorm

import (
	"errors"
	"github.com/caixr9527/zorm/render"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const defaultMaxMemory = 32 << 20

type Context struct {
	W           http.ResponseWriter
	R           *http.Request
	engine      *Engine
	queryParams url.Values
	formParams  url.Values
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
