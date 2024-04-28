package register

import "time"

type Option struct {
	Endpoints   []string
	DialTimeout time.Duration
	ServiceName string
	Host        string
	Port        int
}

type ZRegister interface {
	CreateCli(option Option) error
	RegisterService(serviceName string, host string, port int) error
	GetInstance(serviceName string) (string, error)
	Close() error
}
