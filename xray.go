package libxray

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/xtls/xray-core/common/cmdarg"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer *core.Instance
)

const (
	PingDelayTimeout int = 11000
	PingDelayError   int = 10000
)

func startXray(configFile string) (*core.Instance, error) {
	file := cmdarg.Arg{configFile}
	config, err := core.LoadConfig("json", file)
	if err != nil {
		return nil, err
	}

	server, err := core.New(config)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func initEnv(datDir string) {
	os.Setenv("xray.location.asset", datDir)
}

func RunXray(datDir string, config string) string {
	initEnv(datDir)
	coreServer, err := startXray(config)
	if err != nil {
		return err.Error()
	}

	if err := coreServer.Start(); err != nil {
		return err.Error()
	}

	runtime.GC()
	return ""
}

func StopXray() string {
	if coreServer != nil {
		err := coreServer.Close()
		coreServer = nil
		if err != nil {
			return err.Error()
		}
	}
	return ""
}

func XrayVersion() string {
	return core.Version()
}

func Ping(datDir string, config string, timeout int, url string) string {
	initEnv(datDir)
	server, err := startXray(config)
	if err != nil {
		return fmt.Sprintf("%d:%s", PingDelayError, err)
	}

	if err := server.Start(); err != nil {
		return fmt.Sprintf("%d:%s", PingDelayError, err)
	}
	defer server.Close()

	delay, err := measureDelay(server, time.Second*time.Duration(timeout), url)
	if err != nil {
		return fmt.Sprintf("%d:%s", delay, err)
	}
	return fmt.Sprintf("%d:", delay)
}

func measureDelay(inst *core.Instance, timeout time.Duration, url string) (int64, error) {
	start := time.Now()
	delay, err := coreHTTPRequest(inst, timeout, url)
	if err != nil {
		return int64(delay), err
	}
	return time.Since(start).Milliseconds(), nil
}

func coreHTTPClient(inst *core.Instance, timeout time.Duration) (*http.Client, error) {
	if inst == nil {
		return nil, errors.New("core instance nil")
	}

	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := xnet.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
			if err != nil {
				return nil, err
			}
			return core.Dial(ctx, inst, dest)
		},
	}

	c := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return c, nil
}

func coreHTTPRequest(inst *core.Instance, timeout time.Duration, url string) (int, error) {
	c, err := coreHTTPClient(inst, timeout)
	if err != nil {
		return PingDelayError, err
	}

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := c.Do(req)
	if err != nil {
		return PingDelayTimeout, err
	}
	return resp.StatusCode, nil
}

func CustomUUID(str string) string {
	id, err := uuid.ParseString(str)
	if err != nil {
		return err.Error()
	}
	return id.String()
}
