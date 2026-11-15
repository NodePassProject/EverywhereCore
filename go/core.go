package evcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

const (
	CoreTypeXray    = "xray"
	CoreTypeSingBox = "singbox"
	CoreTypeMihomo  = "mihomo"
)

var (
	mu           sync.Mutex
	coreInstance coreRunner
)

type coreRunner interface {
	stop() error
	suspend()
	resume()
	updateDefaultInterface(name string, index int32, isExpensive, isConstrained bool)
	control(method string, params []byte) (any, error)
}

var errControlUnsupported = errors.New("unsupported control method")

func Version() string { return "Everywhere Core v0.2" }

func StartCore(coreType, configContent string, tunFD, mtu int) error {
	mu.Lock()
	defer mu.Unlock()
	if coreInstance != nil {
		return errors.New("a core is already running")
	}
	if tunFD < 0 {
		return errors.New("invalid tun file descriptor")
	}
	if mtu <= 0 {
		mtu = 1500
	}
	var (
		r   coreRunner
		err error
	)
	switch coreType {
	case CoreTypeXray:
		r, err = startXray(configContent, tunFD, mtu)
	case CoreTypeSingBox:
		r, err = startSingBox(configContent, tunFD, mtu)
	case CoreTypeMihomo:
		r, err = startMihomo(configContent, tunFD, mtu)
	default:
		return fmt.Errorf("unknown core type: %s", coreType)
	}
	if err != nil {
		return errors.New(err.Error())
	}
	coreInstance = r
	return nil
}

func Suspend() error {
	mu.Lock()
	defer mu.Unlock()
	if coreInstance == nil {
		return nil
	}
	coreInstance.suspend()
	return nil
}

func Resume() error {
	mu.Lock()
	defer mu.Unlock()
	if coreInstance == nil {
		return nil
	}
	coreInstance.resume()
	return nil
}

func UpdateDefaultInterface(name string, index int32, isExpensive, isConstrained bool) error {
	mu.Lock()
	defer mu.Unlock()
	if coreInstance == nil {
		return nil
	}
	coreInstance.updateDefaultInterface(name, index, isExpensive, isConstrained)
	return nil
}

func Control(method, params string) (result string, err error) {
	mu.Lock()
	r := coreInstance
	mu.Unlock()
	if r == nil {
		return "", errors.New("no core is running")
	}
	defer func() {
		if p := recover(); p != nil {
			result, err = "", fmt.Errorf("control %s panicked: %v", method, p)
		}
	}()
	value, err := r.control(method, []byte(params))
	if err != nil {
		return "", errors.New(err.Error())
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", errors.New(err.Error())
	}
	return string(data), nil
}

func decodeControlParams(params []byte, v any) error {
	if len(params) == 0 {
		return nil
	}
	if err := json.Unmarshal(params, v); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

func StopAll() error {
	mu.Lock()
	prev := coreInstance
	coreInstance = nil
	mu.Unlock()

	go func() {
		defer func() { _ = recover() }()
		if prev != nil {
			_ = prev.stop()
		}
	}()
	return nil
}
