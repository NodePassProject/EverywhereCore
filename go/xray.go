package evcore

import (
	"os"
	"strconv"

	"github.com/xtls/xray-core/core"

	_ "github.com/xtls/xray-core/main/distro/all"
	_ "github.com/xtls/xray-core/main/json"
)

type xrayRunner struct {
	instance *core.Instance
	ctl      *xrayControl
}

func (x *xrayRunner) stop() error {
	if x.ctl != nil {
		x.ctl.close()
	}
	if x.instance != nil {
		return x.instance.Close()
	}
	return nil
}

func (x *xrayRunner) suspend()                                            {}
func (x *xrayRunner) resume()                                             {}
func (x *xrayRunner) updateDefaultInterface(_ string, _ int32, _, _ bool) {}

func (x *xrayRunner) control(method string, params []byte) (any, error) {
	return x.ctl.handle(method, params)
}

func startXray(configContent string, tunFD, _ int) (coreRunner, error) {
	if err := os.Setenv("xray.tun.fd", strconv.Itoa(tunFD)); err != nil {
		return nil, err
	}
	inst, err := core.StartInstance("json", []byte(configContent))
	if err != nil {
		return nil, err
	}
	return &xrayRunner{instance: inst, ctl: newXrayControl(inst, configContent)}, nil
}
