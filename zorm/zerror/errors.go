package zerror

type ZError struct {
	err    error
	ErrFun ErrorFun
}

func Default() *ZError {
	return &ZError{}
}

func (e *ZError) Error() string {
	return e.err.Error()
}

func (e *ZError) Put(zError *ZError) {
	e.check(zError)
}

func (e *ZError) check(zError *ZError) {
	if zError != nil {
		e.err = zError
		panic(zError)
	}
}

type ErrorFun func(zError *ZError)

func (e *ZError) Result(errorFun ErrorFun) {
	e.ErrFun = errorFun
}

func (e *ZError) ExecResult() {
	e.ErrFun(e)
}
