package zorm

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type HandleFunc func(w http.ResponseWriter, r *http.Request)

type routerGroup struct {
	name             string
	handlerFuncMap   map[string]HandleFunc
	handlerMethodMap map[string][]string
}

func (r *routerGroup) Add(name string, handeFunc HandleFunc) {
	r.handlerFuncMap[name] = handeFunc
}

func (r *routerGroup) Any(name string, handeFunc HandleFunc) {
	r.handlerFuncMap[name] = handeFunc
	r.handlerMethodMap["ANY"] = append(r.handlerMethodMap["ANY"], name)
}

func (r *routerGroup) Get(name string, handeFunc HandleFunc) {
	r.handlerFuncMap[name] = handeFunc
	r.handlerMethodMap[http.MethodGet] = append(r.handlerMethodMap[http.MethodGet], name)
}

func (r *routerGroup) Post(name string, handeFunc HandleFunc) {
	r.handlerFuncMap[name] = handeFunc
	r.handlerMethodMap[http.MethodPost] = append(r.handlerMethodMap[http.MethodPost], name)
}

func (r *routerGroup) Put(name string, handeFunc HandleFunc) {
	r.handlerFuncMap[name] = handeFunc
	r.handlerMethodMap[http.MethodPut] = append(r.handlerMethodMap[http.MethodPut], name)
}

func (r *routerGroup) Delete(name string, handeFunc HandleFunc) {
	r.handlerFuncMap[name] = handeFunc
	r.handlerMethodMap[http.MethodDelete] = append(r.handlerMethodMap[http.MethodDelete], name)
}

type router struct {
	routerGroups []*routerGroup
}

func (r *router) Group(name string) *routerGroup {
	routerGroup := &routerGroup{
		name:             name,
		handlerFuncMap:   make(map[string]HandleFunc),
		handlerMethodMap: make(map[string][]string),
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
		for name, methodHandle := range group.handlerFuncMap {
			path := "/"
			if strings.HasPrefix(name, "/") {
				path = name
			} else {
				path = path + name
			}
			url := "/" + group.name + path
			if r.RequestURI == url {
				routers, ok := group.handlerMethodMap["ANY"]
				if ok {
					for _, routerName := range routers {
						if routerName == name {
							methodHandle(w, r)
							return
						}
					}
				}

				routers, ok = group.handlerMethodMap[method]
				if ok {
					for _, routerName := range routers {
						if routerName == name {
							methodHandle(w, r)
							return
						}
					}
				}
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "%s %s not allowed \n", r.RequestURI, method)
				return
			}
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "%s not found \n", r.RequestURI)
}

func (e *Engine) Run() {
	//for _, group := range e.routerGroups {
	//	for key, value := range group.handlerFuncMap {
	//		path := "/"
	//		if strings.HasPrefix(key, "/") {
	//			path = key
	//		} else {
	//			path = path + key
	//		}
	//		http.HandleFunc("/"+group.name+path, value)
	//	}
	//}
	http.Handle("/", e)
	err := http.ListenAndServe(":8111", nil)
	if err != nil {
		log.Fatal(err)
	}
}
