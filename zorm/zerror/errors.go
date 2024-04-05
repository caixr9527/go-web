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

func (e *ZError) Put(err error) {
	e.check(err)
}

func (e *ZError) check(err error) {
	if err != nil {
		e.err = err
		panic(e)
	}
}

type ErrorFun func(zError *ZError)

func (e *ZError) Result(errorFun ErrorFun) {
	e.ErrFun = errorFun
}

func (e *ZError) ExecResult() {
	e.ErrFun(e)
}
