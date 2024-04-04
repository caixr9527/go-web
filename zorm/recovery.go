package zorm

import (
	"errors"
	"fmt"
	"github.com/caixr9527/zorm/zerror"
	"net/http"
	"runtime"
	"strings"
)

func detailMsg(err any) string {
	var pcs [32]uintptr
	n := runtime.Callers(0, pcs[:])
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[0:n] {
		fn := runtime.FuncForPC(pc)
		file, l := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, l))

	}
	return sb.String()
}

func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				err2 := err.(error)
				if err2 != nil {
					var zError *zerror.ZError
					if errors.As(err2, &zError) {
						zError.ExecResult()
						return
					}
				}

				ctx.Logger.Error(detailMsg(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next(ctx)
	}
}
