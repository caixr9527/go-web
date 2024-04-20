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
	"strings"
	"time"
)

type HttpClient struct {
	client http.Client
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
	return &HttpClient{client: client}
}

func (c HttpClient) Get(url string, args map[string]any) ([]byte, error) {
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

func (c HttpClient) GetRequest(url string, args map[string]any) (*http.Request, error) {
	if args != nil && len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	log.Println(url)
	request, err := http.NewRequest("GET", url, nil)
	return request, err
}

func (c HttpClient) PostRequest(url string, args map[string]any) (*http.Request, error) {

	return http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
}

func (c HttpClient) JsonRequest(url string, args map[string]any) (*http.Request, error) {
	marshal, _ := json.Marshal(args)
	return http.NewRequest("POST", url, bytes.NewBuffer(marshal))
}

func (c HttpClient) PostForm(url string, args map[string]any) ([]byte, error) {

	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

func (c HttpClient) PostJson(url string, args map[string]any) ([]byte, error) {
	marshal, _ := json.Marshal(args)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(marshal))
	if err != nil {
		return nil, err
	}
	return c.responseHandler(request)
}

func (c HttpClient) Response(req *http.Request) ([]byte, error) {
	return c.responseHandler(req)
}

func (c HttpClient) responseHandler(request *http.Request) ([]byte, error) {
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

func (c HttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for k, v := range args {
			params.Set(k, fmt.Sprintf("%v", v))
		}
		return params.Encode()
	}
	return ""
}
