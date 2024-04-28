package register

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func CreateEtcdClient(option Option) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   option.Endpoints,
		DialTimeout: option.DialTimeout,
	})
	return client, err
}

func RegisterService(cli *clientv3.Client, serviceName string, host string, port int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := cli.Put(ctx, serviceName, fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	return nil
}

func GetEtcdInstance(cli *clientv3.Client, serviceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := cli.Get(ctx, serviceName)
	if err != nil {
		return "", err
	}
	kvs := response.Kvs
	return string(kvs[0].Value), nil
}
