package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"runtime"
	"strings"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs": 0,
	"http_host":  "127.0.0.1",
	"http_port":  "8000",
	"grpc_host":  "127.0.0.1",
	"grpc_port":  "8080",
	"redis_host": "127.0.0.1",
	"redis_port": "6379",
}

type DefaultChecker struct {
	flags *pflag.FlagSet
}

func (c *DefaultChecker) checkIfUsingDefault(name string) bool {
	flag := true

	flag = flag && os.Getenv(name) == ""
	//flag = flag && c.flags.Lookup(name) == nil

	return flag
}

var checker *DefaultChecker

func init() {
	rootCmd.Flags().String("http_host", "127.0.0.1", "file server http host")
	rootCmd.Flags().String("http_port", "8000", "file server http port")
	rootCmd.Flags().String("grpc_host", "127.0.0.1", "file server rpc host")
	rootCmd.Flags().String("grpc_port", "8080", "file server rpc port")
	rootCmd.Flags().String("redis_host", "127.0.0.1", "redis host")
	rootCmd.Flags().String("redis_port", "6379", "redis port")

	viper.BindPFlag("http_host", rootCmd.Flags().Lookup("http_host"))
	viper.BindPFlag("http_port", rootCmd.Flags().Lookup("http_port"))
	viper.BindPFlag("grpc_host", rootCmd.Flags().Lookup("grpc_host"))
	viper.BindPFlag("grpc_port", rootCmd.Flags().Lookup("grpc_port"))
	viper.BindPFlag("redis_host", rootCmd.Flags().Lookup("redis_host"))
	viper.BindPFlag("redis_port", rootCmd.Flags().Lookup("redis_port"))

	checker = &DefaultChecker{
		flags: rootCmd.Flags(),
	}
}

var rootCmd = &cobra.Command{
	Use: "rhosusr",
	Run: func(cmd *cobra.Command, args []string) {

		printWelcome()

		for k, v := range configDefaults {
			viper.SetDefault(k, v)
		}

		bindEnvs := []string{
			"http_host", "http_port", "grpc_host", "grpc_port", "redis_host", "redis_port",
		}
		for _, env := range bindEnvs {
			err := viper.BindEnv(env)
			if err != nil {
				logrus.Fatalf("error binding env variable: %v", err)
			}
		}

		if os.Getenv("GOMAXPROCS") == "" {
			if viper.IsSet("gomaxprocs") && viper.GetInt("gomaxprocs") > 0 {
				runtime.GOMAXPROCS(viper.GetInt("gomaxprocs"))
			} else {
				runtime.GOMAXPROCS(runtime.NumCPU())
			}
		}

		v := viper.GetViper()

		if checker.checkIfUsingDefault("http_host") || checker.checkIfUsingDefault("http_port") {
			logrus.Warn("file server http address is not set explicitly")
		}
		if checker.checkIfUsingDefault("grpc_host") || checker.checkIfUsingDefault("grpc_port") {
			logrus.Warn("file server grpc address is not set explicitly")
		}

		httpHost := v.GetString("http_host")
		httpPort := v.GetString("http_port")

		grpcHost := v.GetString("http_host")
		grpcPort := v.GetString("http_port")

		redisConfig, err := getRedisConfig()
		if err != nil {
			logrus.Fatalf("error in redis config: %v", err)
		}

		conf := registry.RegistryConfig{
			HttpHost: httpHost,
			HttpPort: httpPort,
			GrpcHost: grpcHost,
			GrpcPort: grpcPort,

			RedisConfig: *redisConfig,
		}
		r, err := registry.NewRegistry(conf)
		if err != nil {
			logrus.Fatalf("error creating registry instance: %v", err)
		}

		r.Start()
	},
}

func getRedisConfig() (*rhosus_redis.RedisConfig, error) {
	v := viper.GetViper()

	numShards := 0

	hostsConf := v.GetString("redis_host")
	portsConf := v.GetString("redis_port")

	redisTLS := v.GetBool("redis_tls")
	redisTLSSkipVerify := v.GetBool("redis_tls_skip_verify")
	masterNamesConf := v.GetString("redis_master_name")

	password := v.GetString("redis_password")
	db := v.GetInt("redis_db")

	var hosts []string
	if hostsConf != "" {
		hosts = strings.Split(hostsConf, ",")
		if len(hosts) > numShards {
			numShards = len(hosts)
		}
	}

	var ports []string
	if portsConf != "" {
		ports = strings.Split(portsConf, ",")
		if len(ports) > numShards {
			numShards = len(ports)
		}
	}

	var masterNames []string
	if masterNamesConf != "" {
		masterNames = strings.Split(masterNamesConf, ",")
		if len(masterNames) > numShards {
			numShards = len(masterNames)
		}
	}

	if masterNamesConf != "" && len(masterNames) < numShards {
		return nil, fmt.Errorf("redis master name must be set for every Redis shard when Sentinel used")
	}

	if len(hosts) <= 1 {
		newHosts := make([]string, numShards)
		for i := 0; i < numShards; i++ {
			if len(hosts) == 0 {
				newHosts[i] = ""
			} else {
				newHosts[i] = hosts[0]
			}
		}
		hosts = newHosts
	} else if len(hosts) != numShards {
		return nil, fmt.Errorf("malformed sharding configuration: wrong number of redis hosts")
	}

	if len(ports) == 0 {
		newPorts := make([]string, numShards)
		for i := 0; i < numShards; i++ {
			if len(ports) == 0 {
				newPorts[i] = ""
			} else {
				newPorts[i] = ports[0]
			}
		}
		ports = newPorts
	} else if len(ports) != numShards {
		return nil, fmt.Errorf("malformed sharding configuration: wrong number of redis ports")
	}

	if len(masterNames) == 0 {
		newMasterNames := make([]string, numShards)
		for i := 0; i < numShards; i++ {
			newMasterNames[i] = ""
		}
		masterNames = newMasterNames
	}

	passwords := make([]string, numShards)
	for i := 0; i < numShards; i++ {
		passwords[i] = password
	}

	dbs := make([]int, numShards)
	for i := 0; i < numShards; i++ {
		dbs[i] = db
	}

	return &rhosus_redis.RedisConfig{
		Hosts:         hosts,
		Ports:         ports,
		Password:      password,
		UseTLS:        redisTLS,
		MasterName:    masterNamesConf,
		TLSSkipVerify: redisTLSSkipVerify,
		DB:            db,
	}, nil
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus registry")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
