package libxray

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/common/memory"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer *core.Instance
)

const (
	pingDelayTimeout int64 = 11000
	pingDelayError   int64 = 10000
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

func setMaxMemory(maxMemory int64) {
	os.Setenv("XRAY_MEMORY_FORCEFREE", "1")
	memory.InitForceFree(maxMemory)
}

func RunXray(datDir string, config string, maxMemory int64) string {
	initEnv(datDir)
	if maxMemory > 0 {
		setMaxMemory(maxMemory)
	}
	coreServer, err := startXray(config)
	if err != nil {
		return err.Error()
	}

	if err := coreServer.Start(); err != nil {
		return err.Error()
	}

	debug.FreeOSMemory()
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
		return fmt.Sprintf("%d:%s", pingDelayError, err)
	}

	if err := server.Start(); err != nil {
		return fmt.Sprintf("%d:%s", pingDelayError, err)
	}
	defer server.Close()

	return measureDelay(server, time.Second*time.Duration(timeout), url)
}

func measureDelay(inst *core.Instance, timeout time.Duration, url string) string {
	c, err := coreHTTPClient(inst, timeout)
	if err != nil {
		return fmt.Sprintf("%d:%s", pingDelayError, err)
	}
	delaySum := int64(0)
	count := int64(0)
	times := 3
	isValid := false
	lastErr := ""
	for i := 0; i < times; i++ {
		delay, err := coreHTTPRequest(c, url)
		if delay != pingDelayTimeout {
			delaySum += delay
			count += 1
			isValid = true
		} else {
			lastErr = err.Error()
		}
	}
	if !isValid {
		return fmt.Sprintf("%d:%s", pingDelayTimeout, lastErr)
	}
	return fmt.Sprintf("%d:%s", delaySum/count, lastErr)
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

func coreHTTPRequest(c *http.Client, url string) (int64, error) {
	start := time.Now()
	req, _ := http.NewRequest("GET", url, nil)
	_, err := c.Do(req)
	if err != nil {
		return pingDelayTimeout, err
	}
	return time.Since(start).Milliseconds(), nil
}

func CustomUUID(str string) string {
	id, err := uuid.ParseString(str)
	if err != nil {
		return err.Error()
	}
	return id.String()
}

// https://github.com/phayes/freeport/blob/master/freeport.go
// GetFreePort asks the kernel for free open ports that are ready to use.
func GetFreePorts(count int) string {
	var ports []int
	for i := 0; i < count; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return ""
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return ""
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ports)), ":"), "[]")
}
