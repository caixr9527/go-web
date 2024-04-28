package register

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func CreateNacosClient() (naming_client.INamingClient, error) {
	// todo 可以做成单例
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			"127.0.0.1",
			8848,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
	}

	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		})
	if err != nil {
		return nil, err
	}
	return client, err
}

// todo 可以优化，提取配置，自动获取ip
func Register(client naming_client.INamingClient, serviceName string, ip string, port uint64) error {
	_, err := client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "shanghai"},
		//ClusterName: "cluster-a",
		//GroupName:   "group-a",
	})
	return err
}

func GetInstance(client naming_client.INamingClient, serviceName string) (string, uint64, error) {
	instance, err := client.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		//ClusterName: "cluster-a",
		//GroupName:   "group-a",
	})
	if err != nil {
		return "", 0, err
	}
	return instance.Ip, instance.Port, nil
}

type NacosRegister struct {
	client naming_client.INamingClient
}
