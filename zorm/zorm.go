package zorm

import (
	"errors"
	"fmt"
	"github.com/caixr9527/zorm/config"
	"github.com/caixr9527/zorm/gateway"
	zormlog "github.com/caixr9527/zorm/log"
	"github.com/caixr9527/zorm/render"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

const ANY = "ANY"

type HandlerFunc func(ctx *Context)

type MiddlewareFunc func(handlerFunc HandlerFunc) HandlerFunc

type routerGroup struct {
	name               string
	handlerFuncMap     map[string]map[string]HandlerFunc
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc
	handlerMethodMap   map[string][]string
	treeNode           *treeNode
	middlewares        []MiddlewareFunc
}

func (r *routerGroup) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
}

func (r *routerGroup) methodHandle(name string, method string, h HandlerFunc, ctx *Context) {
	// group pre
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			h = middlewareFunc(h)
		}
	}
	// router level
	middlewareFuncs := r.middlewaresFuncMap[name][method]
	if middlewareFuncs != nil {
		for _, middlewareFunc := range middlewareFuncs {
			h = middlewareFunc(h)
		}
	}
	h(ctx)
}

func (r *routerGroup) handle(name string, method string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	_, ok := r.handlerFuncMap[name]
	if !ok {
		r.handlerFuncMap[name] = make(map[string]HandlerFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handlerFuncMap[name][method]
	if ok {
		panic("Duplicate routing [" + name + "]")
	}
	r.handlerFuncMap[name][method] = handleFunc
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...)
	r.treeNode.Put(name)
}

func (r *routerGroup) Any(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, ANY, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Get(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handleFunc, middlewareFunc...)

}

func (r *routerGroup) Post(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handleFunc, middlewareFunc...)

}

func (r *routerGroup) Put(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handleFunc, middlewareFunc...)

}

func (r *routerGroup) Delete(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Patch(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handleFunc, middlewareFunc...)
}

func (r *routerGroup) Options(name string, handleFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handleFunc, middlewareFunc...)
}

type router struct {
	routerGroups []*routerGroup
	engine       *Engine
}

func (r *router) Group(name string) *routerGroup {
	routerGroup := &routerGroup{
		name:               name,
		handlerFuncMap:     make(map[string]map[string]HandlerFunc),
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
		handlerMethodMap:   make(map[string][]string),
		treeNode:           &treeNode{name: "/", children: make([]*treeNode, 0)},
	}
	routerGroup.Use(r.engine.middles...)
	r.routerGroups = append(r.routerGroups, routerGroup)
	return routerGroup
}

type ErrorHandler func(err error) (int, any)

type Engine struct {
	router
	funcMap          template.FuncMap
	HTMLRender       render.HTMLRender
	pool             sync.Pool
	Logger           *zormlog.Logger
	middles          []MiddlewareFunc
	errorHandler     ErrorHandler
	OpenGateway      bool
	gatewayConfigs   []gateway.GWConfig
	gatewayTreeNode  *gateway.TreeNode
	gatewayConfigMap map[string]gateway.GWConfig
}

func New() *Engine {
	engine := &Engine{
		router:           router{},
		gatewayTreeNode:  &gateway.TreeNode{Name: "/", Children: make([]*gateway.TreeNode, 0)},
		gatewayConfigMap: make(map[string]gateway.GWConfig),
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}

func Default() *Engine {
	engine := New()
	engine.Logger = zormlog.Default()
	logPath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logPath.(string))
	} else {
		engine.Logger.SetLogPath("./log")
	}
	engine.Use(Logging, Recovery)
	engine.router.engine = engine
	return engine
}

func (e *Engine) allocateContext() any {
	return &Context{engine: e}
}

func (e *Engine) SetGatewayConfig(configs []gateway.GWConfig) {
	for _, v := range configs {
		e.gatewayTreeNode.Put(v.Path, v.Name)
		e.gatewayConfigMap[v.Name] = v
	}
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHTMLTemplate(t)
}

func (e *Engine) LoadTemplateConfig() error {
	pattern, ok := config.Conf.Template["pattern"]
	if !ok {
		// todo 抛异常 打日志
		return errors.New("template pattern config not found")
	}
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern.(string)))
	e.SetHTMLTemplate(t)
	return nil
}

func (e *Engine) SetHTMLTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: t}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.R = r
	ctx.Logger = e.Logger
	e.httpRequestHandler(ctx, w, r)
	e.pool.Put(ctx)
}

func (e *Engine) httpRequestHandler(ctx *Context, w http.ResponseWriter, r *http.Request) {
	if e.OpenGateway {
		path := r.URL.Path
		node := e.gatewayTreeNode.Get(path)
		if node == nil {
			ctx.W.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(ctx.W, ctx.R.RequestURI+" not found")
			return
		}
		gwConfig := e.gatewayConfigMap[node.GwName]
		gwConfig.Header(ctx.R)
		target, err := url.Parse(fmt.Sprintf("http://%s:%d%s", gwConfig.Host, gwConfig.Port, path))
		if err != nil {
			ctx.W.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(ctx.W, err.Error())
			return
		}
		director := func(req *http.Request) {
			req.Host = target.Host
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.Scheme = target.Scheme
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
		}
		response := func(response *http.Response) error {
			// todo
			log.Println("响应修改")
			return nil
		}
		handler := func(writer http.ResponseWriter, request *http.Request, err error) {
			log.Println("错误处理")
		}
		proxy := httputil.ReverseProxy{
			Director:       director,
			ModifyResponse: response,
			ErrorHandler:   handler,
		}
		proxy.ServeHTTP(w, r)
		return
	}
	method := r.Method
	for _, group := range e.routerGroups {
		routerName := SubStringLast(r.URL.Path, "/"+group.name)
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd {

			handle, ok := group.handlerFuncMap[node.routerName][ANY]
			if ok {
				group.methodHandle(node.routerName, ANY, handle, ctx)
				return
			}
			handle, ok = group.handlerFuncMap[node.routerName][method]
			if ok {
				group.methodHandle(node.routerName, method, handle, ctx)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "%s %s not allowed \n", r.RequestURI, method)
			return
		}

	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "%s not found \n", r.RequestURI)
}

func (e *Engine) Run(addr string) {
	http.Handle("/", e)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Engine) Use(middles ...MiddlewareFunc) {
	e.middles = append(e.middles, middles...)
}

func (e *Engine) RegisterErrorHandler(handler ErrorHandler) {
	e.errorHandler = handler
}

func (e *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Engine) Handler() http.Handler {
	return e
}
