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

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer *core.Instance
)

func startXray(configFile string) (*core.Instance, error) {
	config, err := core.LoadConfig("json", configFile)
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
		return fmt.Sprintf("0:%s", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Sprintf("0:%s", err)
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
	_, err := coreHTTPRequest(inst, timeout, url)
	if err != nil {
		return 0, err
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
		return 0, err
	}

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := c.Do(req)
	if err != nil {
		return -1, err
	}
	return resp.StatusCode, nil
}
