package zorm

import (
	"fmt"
	"log"
	"net/http"
)

const ANY = "ANY"

type HandleFunc func(ctx *Context)

type routerGroup struct {
	name             string
	handlerFuncMap   map[string]map[string]HandleFunc
	handlerMethodMap map[string][]string
	treeNode         *treeNode
}

func (r *routerGroup) handle(name string, method string, handleFunc HandleFunc) {
	_, ok := r.handlerFuncMap[name]
	if !ok {
		r.handlerFuncMap[name] = make(map[string]HandleFunc)
	}
	_, ok = r.handlerFuncMap[name][method]
	if ok {
		panic("Duplicate routing [" + name + "]")
	}
	r.handlerFuncMap[name][method] = handleFunc

	r.treeNode.Put(name)
}

func (r *routerGroup) Any(name string, handleFunc HandleFunc) {
	r.handle(name, ANY, handleFunc)
}

func (r *routerGroup) Get(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodGet, handleFunc)

}

func (r *routerGroup) Post(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodPost, handleFunc)

}

func (r *routerGroup) Put(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodPut, handleFunc)

}

func (r *routerGroup) Delete(name string, handleFunc HandleFunc) {
	r.handle(name, http.MethodDelete, handleFunc)
}

type router struct {
	routerGroups []*routerGroup
}

func (r *router) Group(name string) *routerGroup {
	routerGroup := &routerGroup{
		name:             name,
		handlerFuncMap:   make(map[string]map[string]HandleFunc),
		handlerMethodMap: make(map[string][]string),
		treeNode:         &treeNode{name: "/", children: make([]*treeNode, 0)},
	}
	r.routerGroups = append(r.routerGroups, routerGroup)
	return routerGroup
}

type Engine struct {
	router
}

func New() *Engine {
	return &Engine{
		router: router{},
	}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	for _, group := range e.routerGroups {
		routerName := SubStringLast(r.RequestURI, "/"+group.name)
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			ctx := &Context{
				W: w,
				R: r,
			}
			handle, ok := group.handlerFuncMap[node.routerName][ANY]
			if ok {
				handle(ctx)
				return
			}
			handle, ok = group.handlerFuncMap[node.routerName][method]
			if ok {
				handle(ctx)
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

func (e *Engine) Run() {
	http.Handle("/", e)
	err := http.ListenAndServe(":8111", nil)
	if err != nil {
		log.Fatal(err)
	}
}
