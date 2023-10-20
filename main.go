package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"github.com/temporalio/cli/server"
	cliconfig "github.com/temporalio/tctl-kit/pkg/config"
	uiserver "github.com/temporalio/ui-server/v2/server"
	uiconfig "github.com/temporalio/ui-server/v2/server/config"
	uiserveroptions "github.com/temporalio/ui-server/v2/server/server_options"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/temporal"
)

func main() {
	ctx := context.Background()

	serverPort := 7233
	uiPort := 8088
	httpPort := 0    // disabled
	metricsPort := 0 // disabled
	ip := "0.0.0.0"

	pragmas, err := getPragmaMap([]string{})
	if err != nil {
		panic(err)
	}
	interruptChan := make(chan interface{}, 1)
	go func() {
		if doneChan := ctx.Done(); doneChan != nil {
			s := <-doneChan
			interruptChan <- s
		} else {
			s := <-temporal.InterruptCh()
			interruptChan <- s
		}
	}()

	opts := []server.ServerOption{
		server.WithDynamicPorts(),
		server.WithFrontendPort(serverPort),
		server.WithFrontendHTTPPort(httpPort),
		server.WithMetricsPort(metricsPort),
		server.WithFrontendIP(ip),
		server.WithNamespaces("default"),
		server.WithSQLitePragmas(pragmas),
		server.WithUpstreamOptions(
			temporal.InterruptOn(interruptChan),
		),
		server.WithBaseConfig(&config.Config{}),
	}

	frontendAddr := fmt.Sprintf("%s:%d", ip, serverPort)

	uiBaseCfg := &uiconfig.Config{
		Host:                ip,
		Port:                uiPort,
		TemporalGRPCAddress: frontendAddr,
		EnableUI:            true,
		EnableOpenAPI:       true,
	}
	opts = append(opts, server.WithUI(uiserver.NewServer(uiserveroptions.WithConfigProvider(uiBaseCfg))))

	opts = append(opts, server.WithPersistenceDisabled())
	if clusterCfg, err := cliconfig.NewConfig("temporalio", "version-info"); err == nil {
		defaultEnv := "default"
		clusterIDKey := "cluster-id"

		clusterID, _ := clusterCfg.EnvProperty(defaultEnv, clusterIDKey)

		if clusterID == "" {
			// fallback to generating a new cluster Id in case of errors or empty value
			clusterID = uuid.New()
			clusterCfg.SetEnvProperty(defaultEnv, clusterIDKey, clusterID)
		}

		opts = append(opts, server.WithCustomClusterID(clusterID))
	}

	s, err := server.NewServer(opts...)
	if err != nil {
		panic(err)
	}

	if err := s.Start(); err != nil {
		panic(err)
	}
}

func getPragmaMap(input []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, pragma := range input {
		vals := strings.Split(pragma, "=")
		if len(vals) != 2 {
			return nil, fmt.Errorf("ERROR: pragma statements must be in KEY=VALUE format, got %q", pragma)
		}
		result[vals[0]] = vals[1]
	}
	return result, nil
}
