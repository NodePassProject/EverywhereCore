package evcore

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"sync"
	"syscall"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	tun "github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/logger"
	"github.com/sagernet/sing/common/x/list"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
)

type singBoxRunner struct {
	box          *box.Box
	pauseManager pause.Manager
	platform     *singBoxPlatform
}

func (s *singBoxRunner) stop() error {
	if s.box != nil {
		return s.box.Close()
	}
	return nil
}

func (s *singBoxRunner) suspend() {
	if s.pauseManager != nil && !s.pauseManager.IsDevicePaused() {
		s.pauseManager.DevicePause()
	}
}

func (s *singBoxRunner) resume() {
	if s.pauseManager != nil && s.pauseManager.IsDevicePaused() {
		s.pauseManager.DeviceWake()
	}
}

func (s *singBoxRunner) updateDefaultInterface(name string, index int32, isExpensive, isConstrained bool) {
	if s.platform == nil || s.platform.monitor == nil {
		return
	}
	var nm adapter.NetworkManager
	if s.box != nil {
		nm = s.box.Network()
	}
	s.platform.monitor.updateDefault(nm, name, index, isExpensive, isConstrained)
}

func (s *singBoxRunner) control(method string, _ []byte) (any, error) {
	return nil, fmt.Errorf("%w: %s", errControlUnsupported, method)
}

func startSingBox(configContent string, tunFD, _ int) (coreRunner, error) {
	ctx := include.Context(context.Background())

	ctx = pause.WithDefaultManager(ctx)
	pauseManager := service.FromContext[pause.Manager](ctx)

	pi := newSingBoxPlatform(tunFD)
	ctx = service.ContextWith[adapter.PlatformInterface](ctx, pi)

	options, err := json.UnmarshalExtendedContext[option.Options](ctx, []byte(configContent))
	if err != nil {
		return nil, err
	}
	b, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		return nil, err
	}
	if err := b.Start(); err != nil {
		_ = b.Close()
		return nil, err
	}
	return &singBoxRunner{box: b, pauseManager: pauseManager, platform: pi}, nil
}

type singBoxPlatform struct {
	tunFD       int
	myAddresses []netip.Addr
	monitor     *singBoxInterfaceMonitor
}

func newSingBoxPlatform(tunFD int) *singBoxPlatform {
	return &singBoxPlatform{
		tunFD:   tunFD,
		monitor: newSingBoxInterfaceMonitor(),
	}
}

func (p *singBoxPlatform) Initialize(_ adapter.NetworkManager) error { return nil }

func (p *singBoxPlatform) UsePlatformAutoDetectInterfaceControl() bool { return false }
func (p *singBoxPlatform) AutoDetectInterfaceControl(_ int) error      { return nil }

func (p *singBoxPlatform) UsePlatformInterface() bool { return true }

func (p *singBoxPlatform) OpenInterface(options *tun.Options, _ option.TunPlatformOptions) (tun.Tun, error) {
	dupFd, err := syscall.Dup(p.tunFD)
	if err != nil {
		return nil, err
	}
	options.FileDescriptor = dupFd
	for _, prefix := range options.Inet4Address {
		p.myAddresses = append(p.myAddresses, prefix.Addr())
	}
	for _, prefix := range options.Inet6Address {
		p.myAddresses = append(p.myAddresses, prefix.Addr())
	}
	return tun.New(*options)
}

func (p *singBoxPlatform) ProcessPlatformOptions(_ option.TunPlatformOptions) error {
	return nil
}

func (p *singBoxPlatform) UsePlatformDefaultInterfaceMonitor() bool { return true }
func (p *singBoxPlatform) CreateDefaultInterfaceMonitor(_ logger.Logger) tun.DefaultInterfaceMonitor {
	return p.monitor
}

func (p *singBoxPlatform) UsePlatformNetworkInterfaces() bool { return false }
func (p *singBoxPlatform) NetworkInterfaces() ([]adapter.NetworkInterface, error) {
	return nil, os.ErrInvalid
}

func (p *singBoxPlatform) UnderNetworkExtension() bool              { return true }
func (p *singBoxPlatform) NetworkExtensionIncludeAllNetworks() bool { return false }

func (p *singBoxPlatform) ClearDNSCache()                       {}
func (p *singBoxPlatform) RequestPermissionForWIFIState() error { return nil }

func (p *singBoxPlatform) ReadWIFIState(_ context.Context) adapter.WIFIState {
	return adapter.WIFIState{}
}

func (p *singBoxPlatform) UsePlatformConnectionOwnerFinder() bool { return false }
func (p *singBoxPlatform) FindConnectionOwner(_ *adapter.FindConnectionOwnerRequest) (*adapter.ConnectionOwner, error) {
	return nil, os.ErrInvalid
}

func (p *singBoxPlatform) UsePlatformWIFIMonitor() bool                   { return false }
func (p *singBoxPlatform) UsePlatformNotification() bool                  { return false }
func (p *singBoxPlatform) SendNotification(_ *adapter.Notification) error { return nil }

func (p *singBoxPlatform) CancelNotification(_ string, _ int32) error { return nil }

func (p *singBoxPlatform) MyInterfaceAddress() []netip.Addr { return p.myAddresses }

func (p *singBoxPlatform) UsePlatformNeighborResolver() bool { return false }
func (p *singBoxPlatform) StartNeighborMonitor(_ adapter.NeighborUpdateListener) error {
	return os.ErrInvalid
}
func (p *singBoxPlatform) CloseNeighborMonitor(_ adapter.NeighborUpdateListener) error { return nil }

func (p *singBoxPlatform) UsePlatformShell() bool    { return false }
func (p *singBoxPlatform) CheckPlatformShell() error { return nil }
func (p *singBoxPlatform) OpenShellSession(_ *adapter.PlatformUser, _ string, _ []string, _ string, _, _ int32) (adapter.ShellSession, error) {
	return nil, os.ErrInvalid
}
func (p *singBoxPlatform) LookupUser(_ string) (*adapter.PlatformUser, error) {
	return nil, os.ErrInvalid
}
func (p *singBoxPlatform) LookupSFTPServer() (string, error)     { return "", os.ErrInvalid }
func (p *singBoxPlatform) ReadSystemSSHHostKey() ([]byte, error) { return nil, os.ErrInvalid }
func (p *singBoxPlatform) TailscaleHostname() string             { return "" }

func (p *singBoxPlatform) UsePlatformBridge() bool { return false }
func (p *singBoxPlatform) CreateBridge(_ adapter.BridgeOptions) (adapter.BridgeSession, error) {
	return nil, os.ErrInvalid
}

type singBoxInterfaceMonitor struct {
	access       sync.Mutex
	current      *control.Interface
	myInterfaces []string
	callbacks    list.List[tun.DefaultInterfaceUpdateCallback]
}

func newSingBoxInterfaceMonitor() *singBoxInterfaceMonitor {
	return &singBoxInterfaceMonitor{}
}

func (m *singBoxInterfaceMonitor) Start() error             { return nil }
func (m *singBoxInterfaceMonitor) Close() error             { return nil }
func (m *singBoxInterfaceMonitor) OverrideAndroidVPN() bool { return false }
func (m *singBoxInterfaceMonitor) AndroidVPNEnabled() bool  { return false }

func (m *singBoxInterfaceMonitor) DefaultInterface() *control.Interface {
	m.access.Lock()
	defer m.access.Unlock()
	return m.current
}

func (m *singBoxInterfaceMonitor) RegisterCallback(callback tun.DefaultInterfaceUpdateCallback) *list.Element[tun.DefaultInterfaceUpdateCallback] {
	m.access.Lock()
	defer m.access.Unlock()
	return m.callbacks.PushBack(callback)
}

func (m *singBoxInterfaceMonitor) UnregisterCallback(element *list.Element[tun.DefaultInterfaceUpdateCallback]) {
	m.access.Lock()
	defer m.access.Unlock()
	m.callbacks.Remove(element)
}

func (m *singBoxInterfaceMonitor) RegisterMyInterface(interfaceName string) {
	m.access.Lock()
	defer m.access.Unlock()
	m.myInterfaces = append(m.myInterfaces, interfaceName)
}

func (m *singBoxInterfaceMonitor) MyInterfaces() []string {
	m.access.Lock()
	defer m.access.Unlock()
	return m.myInterfaces
}

func (m *singBoxInterfaceMonitor) updateDefault(nm adapter.NetworkManager, name string, index int32, _ bool, _ bool) {
	if nm != nil {
		_ = nm.UpdateInterfaces()
	}

	var newInterface *control.Interface
	if index >= 0 {
		if nm != nil {
			if iif, err := nm.InterfaceFinder().ByIndex(int(index)); err == nil {
				newInterface = iif
			}
		}
		if newInterface == nil {
			newInterface = &control.Interface{Name: name, Index: int(index)}
		}
	}

	m.access.Lock()
	old := m.current
	m.current = newInterface
	if newInterface != nil && old != nil && old.Index == newInterface.Index && old.Name == newInterface.Name {
		m.access.Unlock()
		return
	}
	callbacks := make([]tun.DefaultInterfaceUpdateCallback, 0, m.callbacks.Len())
	for el := m.callbacks.Front(); el != nil; el = el.Next() {
		callbacks = append(callbacks, el.Value)
	}
	m.access.Unlock()

	for _, callback := range callbacks {
		callback(newInterface, 0)
	}
}
