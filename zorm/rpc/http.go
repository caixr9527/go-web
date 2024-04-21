package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type HttpClient struct {
	client     http.Client
	serviceMap map[string]ZService
}

func NewHttpClient() *HttpClient {
	client := http.Client{
		Timeout: time.Duration(3) * time.Second,
		Transport: &http.Transport{
			MaxConnsPerHost:       100,
			MaxIdleConnsPerHost:   5,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &HttpClient{client: client, serviceMap: make(map[string]ZService)}
}

func (c *HttpClient) Get(url string, args map[string]any) ([]byte, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	log.Println(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

func (c *HttpClient) GetRequest(url string, args map[string]any) (*http.Request, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	log.Println(url)
	request, err := http.NewRequest("GET", url, nil)
	return request, err
}

func (c *HttpClient) PostRequest(url string, args map[string]any) (*http.Request, error) {

	return http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
}

func (c *HttpClient) JsonRequest(url string, args map[string]any) (*http.Request, error) {
	marshal, _ := json.Marshal(args)
	return http.NewRequest("POST", url, bytes.NewBuffer(marshal))
}

func (c *HttpClient) PostForm(url string, args map[string]any) ([]byte, error) {

	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

func (c *HttpClient) PostJson(url string, args map[string]any) ([]byte, error) {
	marshal, _ := json.Marshal(args)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(marshal))
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

func (c *HttpClient) Response(req *http.Request) ([]byte, error) {
	return c.responseHandler(req)
}

func (c *HttpClient) responseHandler(request *http.Request) ([]byte, error) {
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("response status is %d", response.StatusCode))
	}
	reader := bufio.NewReader(response.Body)
	defer response.Body.Close()
	var buf = make([]byte, 127)
	var body []byte
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	return body, nil
}

func (c *HttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for k, v := range args {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		return params.Encode()
	}
	return ""
}

type HttpConfig struct {
	Protocol string
	Host     string
	Port     int
}

const (
	HTTP  = "http"
	HTTPS = "https"
)

const (
	GET       = "GET"
	POST_FORM = "POST_FORM"
	POST_JSON = "POST_JSON"
)

type ZService interface {
	Env() HttpConfig
}

func (c *HttpClient) RegisterHttpService(name string, service ZService) {
	c.serviceMap[name] = service
}
func (c *HttpClient) Do(service string, method string) ZService {
	zService, ok := c.serviceMap[service]
	if !ok {
		panic(errors.New("service not found"))
	}
	t := reflect.TypeOf(zService)
	v := reflect.ValueOf(zService)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("service not pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	fieldIndex := -1
	for i := 0; i < tVar.NumField(); i++ {
		name := tVar.Field(i).Name
		if name == method {
			fieldIndex = i
			break
		}
	}
	if fieldIndex == -1 {
		panic(errors.New("method not found"))
	}
	tag := tVar.Field(fieldIndex).Tag
	rpcInfo := tag.Get("zrpc")
	if rpcInfo == "" {
		panic(errors.New("not rpc tag"))
	}
	split := strings.Split(rpcInfo, ",")
	if len(split) != 2 {
		panic(errors.New("tag rpc not valid"))
	}
	methodType := split[0]
	path := split[1]
	httpConfig := zService.Env()

	f := func(args map[string]any) ([]byte, error) {
		if methodType == GET {
			return c.Get(httpConfig.Prefix()+path, args)
		}
		if methodType == POST_FORM {
			return c.PostForm(httpConfig.Prefix()+path, args)
		}
		if methodType == POST_JSON {
			return c.PostForm(httpConfig.Prefix()+path, args)
		}
		return nil, errors.New("no match method type")
	}
	fValue := reflect.ValueOf(f)
	vVar.Field(fieldIndex).Set(fValue)
	return zService
}

func (c HttpConfig) Prefix() string {
	if c.Protocol == "" {
		c.Protocol = HTTP
	}
	switch c.Protocol {
	case HTTP:
		return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
	case HTTPS:
		return fmt.Sprintf("https://%s:%d", c.Host, c.Port)
	}
	return ""
}
