package evcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	applog "github.com/xtls/xray-core/app/log"
	"github.com/xtls/xray-core/app/observatory"
	commonlog "github.com/xtls/xray-core/common/log"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/extension"
	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/features/routing"
	featurestats "github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/transport/internet/tagged"

	_ "github.com/xtls/xray-core/transport/internet/tagged/taggedimpl"
)

const (
	xrayDefaultProbeURL     = "https://www.gstatic.com/generate_204"
	xrayDefaultProbeTimeout = 5 * time.Second
	xrayMaxProbeTimeout     = 30 * time.Second
	xrayProbeConcurrency    = 8

	xrayLogCapacity    = 500
	xrayLogIdleTimeout = 15 * time.Second
)

type xrayStringList []string

func (l *xrayStringList) UnmarshalJSON(data []byte) error {
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*l = list
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*l = nil
	for _, part := range strings.Split(single, ",") {
		if part = strings.TrimSpace(part); part != "" {
			*l = append(*l, part)
		}
	}
	return nil
}

type xrayConfigView struct {
	Inbounds []struct {
		Tag      string `json:"tag"`
		Protocol string `json:"protocol"`
	} `json:"inbounds"`
	Outbounds []struct {
		Tag      string `json:"tag"`
		Protocol string `json:"protocol"`
	} `json:"outbounds"`
	Routing *struct {
		DomainStrategy string            `json:"domainStrategy"`
		Rules          []json.RawMessage `json:"rules"`
		Balancers      []struct {
			Tag      string         `json:"tag"`
			Selector xrayStringList `json:"selector"`
			Strategy struct {
				Type string `json:"type"`
			} `json:"strategy"`
			FallbackTag string `json:"fallbackTag"`
		} `json:"balancers"`
	} `json:"routing"`
}

type xrayProbeResult struct {
	delay    int64
	testedAt int64
	err      string
}

type xrayControl struct {
	instance *core.Instance
	ctx      context.Context
	started  time.Time
	view     xrayConfigView

	router      routing.Router
	dispatcher  routing.Dispatcher
	ohm         outbound.Manager
	stats       featurestats.Manager
	observatory extension.Observatory

	probeMu sync.Mutex
	probes  map[string]xrayProbeResult

	logs       *xrayLogBuffer
	logHandler commonlog.Handler
}

func newXrayControl(inst *core.Instance, configContent string) *xrayControl {
	c := &xrayControl{
		instance: inst,
		ctx:      context.WithValue(context.Background(), core.XrayKey(1), inst),
		started:  time.Now(),
		probes:   make(map[string]xrayProbeResult),
		logs:     newXrayLogBuffer(xrayLogCapacity),
	}
	_ = json.Unmarshal([]byte(configContent), &c.view)

	if f, ok := inst.GetFeature(routing.RouterType()).(routing.Router); ok {
		c.router = f
	}
	if f, ok := inst.GetFeature(routing.DispatcherType()).(routing.Dispatcher); ok {
		c.dispatcher = f
	}
	if f, ok := inst.GetFeature(outbound.ManagerType()).(outbound.Manager); ok {
		c.ohm = f
	}
	if f, ok := inst.GetFeature(featurestats.ManagerType()).(featurestats.Manager); ok {
		c.stats = f
	}
	if f, ok := inst.GetFeature(extension.ObservatoryType()).(extension.Observatory); ok {
		c.observatory = f
	}

	if h, ok := inst.GetFeature((*applog.Instance)(nil)).(commonlog.Handler); ok {
		c.logHandler = h
	}
	commonlog.RegisterHandler(&xrayLogTee{next: c.logHandler, buf: c.logs})
	return c
}

func (c *xrayControl) close() {
	if c.logHandler != nil {
		commonlog.RegisterHandler(c.logHandler)
	}
}

func (c *xrayControl) handle(method string, params []byte) (any, error) {
	switch method {
	case "info":
		return c.info(), nil
	case "proxies":
		return c.proxies(), nil
	case "stats":
		return c.statsSnapshot(), nil
	case "select":
		var p struct {
			Balancer string `json:"balancer"`
			Target   string `json:"target"`
		}
		if err := decodeControlParams(params, &p); err != nil {
			return nil, err
		}
		if err := c.selectTarget(p.Balancer, p.Target); err != nil {
			return nil, err
		}
		return struct{}{}, nil
	case "delay":
		var p struct {
			Tags    []string `json:"tags"`
			URL     string   `json:"url"`
			Timeout int64    `json:"timeout"`
		}
		if err := decodeControlParams(params, &p); err != nil {
			return nil, err
		}
		return c.delay(p.Tags, p.URL, time.Duration(p.Timeout)*time.Millisecond)
	case "logs":
		var p struct {
			After int64  `json:"after"`
			Limit int    `json:"limit"`
			Level string `json:"level"`
		}
		if err := decodeControlParams(params, &p); err != nil {
			return nil, err
		}
		return c.logs.read(p.After, p.Limit, p.Level), nil
	default:
		return nil, fmt.Errorf("%w: %s", errControlUnsupported, method)
	}
}

type xrayTaggedItem struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
}

type xrayInfo struct {
	Version        string            `json:"version"`
	Started        int64             `json:"started"`
	StatsEnabled   bool              `json:"statsEnabled"`
	Observatory    bool              `json:"observatory"`
	DomainStrategy string            `json:"domainStrategy"`
	Inbounds       []xrayTaggedItem  `json:"inbounds"`
	Outbounds      []xrayTaggedItem  `json:"outbounds"`
	Rules          []json.RawMessage `json:"rules"`
}

func (c *xrayControl) info() xrayInfo {
	info := xrayInfo{
		Version:      core.Version(),
		Started:      c.started.Unix(),
		StatsEnabled: c.counterVisitor() != nil,
		Observatory:  c.observatory != nil,
		Inbounds:     []xrayTaggedItem{},
		Outbounds:    []xrayTaggedItem{},
		Rules:        []json.RawMessage{},
	}
	for _, in := range c.view.Inbounds {
		info.Inbounds = append(info.Inbounds, xrayTaggedItem{Tag: in.Tag, Protocol: in.Protocol})
	}
	for _, out := range c.view.Outbounds {
		info.Outbounds = append(info.Outbounds, xrayTaggedItem{Tag: out.Tag, Protocol: out.Protocol})
	}
	if c.view.Routing != nil {
		info.DomainStrategy = c.view.Routing.DomainStrategy
		if c.view.Routing.Rules != nil {
			info.Rules = c.view.Routing.Rules
		}
	}
	return info
}

type xrayOutboundState struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Delay    int64  `json:"delay"`
	TestedAt int64  `json:"testedAt"`
	Error    string `json:"error,omitempty"`
}

type xrayBalancerState struct {
	Tag         string   `json:"tag"`
	Strategy    string   `json:"strategy"`
	FallbackTag string   `json:"fallbackTag,omitempty"`
	Candidates  []string `json:"candidates"`
	Override    string   `json:"override"`
	Targets     []string `json:"targets"`
}

type xrayProxies struct {
	Default   string              `json:"default"`
	Outbounds []xrayOutboundState `json:"outbounds"`
	Balancers []xrayBalancerState `json:"balancers"`
}

func (c *xrayControl) proxies() xrayProxies {
	result := xrayProxies{
		Outbounds: []xrayOutboundState{},
		Balancers: []xrayBalancerState{},
	}
	if c.ohm != nil {
		if h := c.ohm.GetDefaultHandler(); h != nil {
			result.Default = h.Tag()
		}
	}

	observed := c.observed()
	c.probeMu.Lock()
	for _, out := range c.view.Outbounds {
		if out.Tag == "" {
			continue
		}
		state := xrayOutboundState{Tag: out.Tag, Protocol: out.Protocol}
		if p, ok := c.probes[out.Tag]; ok {
			state.Delay, state.TestedAt, state.Error = p.delay, p.testedAt, p.err
		}
		if o, ok := observed[out.Tag]; ok && o.testedAt >= state.TestedAt {
			state.Delay, state.TestedAt, state.Error = o.delay, o.testedAt, o.err
		}
		result.Outbounds = append(result.Outbounds, state)
	}
	c.probeMu.Unlock()

	if c.view.Routing == nil {
		return result
	}
	overrider, _ := c.router.(routing.BalancerOverrider)
	principle, _ := c.router.(routing.BalancerPrincipleTarget)
	for _, b := range c.view.Routing.Balancers {
		if b.Tag == "" {
			continue
		}
		state := xrayBalancerState{
			Tag:         b.Tag,
			Strategy:    strings.ToLower(b.Strategy.Type),
			FallbackTag: b.FallbackTag,
			Candidates:  c.candidates(b.Selector),
			Targets:     []string{},
		}
		if state.Strategy == "" {
			state.Strategy = "random"
		}
		if overrider != nil {
			if target, err := overrider.GetOverrideTarget(b.Tag); err == nil {
				state.Override = target
			}
		}
		if principle != nil {
			if targets, err := principle.GetPrincipleTarget(b.Tag); err == nil && targets != nil {
				state.Targets = targets
			}
		}
		result.Balancers = append(result.Balancers, state)
	}
	return result
}

func (c *xrayControl) candidates(selectors []string) []string {
	if hs, ok := c.ohm.(outbound.HandlerSelector); ok {
		return append([]string{}, hs.Select(selectors)...)
	}
	tags := []string{}
	for _, out := range c.view.Outbounds {
		for _, selector := range selectors {
			if out.Tag != "" && strings.HasPrefix(out.Tag, selector) {
				tags = append(tags, out.Tag)
				break
			}
		}
	}
	sort.Strings(tags)
	return tags
}

func (c *xrayControl) observed() map[string]xrayProbeResult {
	result := map[string]xrayProbeResult{}
	if c.observatory == nil {
		return result
	}
	message, err := c.observatory.GetObservation(c.ctx)
	if err != nil {
		return result
	}
	observation, ok := message.(*observatory.ObservationResult)
	if !ok {
		return result
	}
	for _, status := range observation.GetStatus() {
		if status == nil || status.GetLastTryTime() == 0 {
			continue
		}
		r := xrayProbeResult{testedAt: status.GetLastTryTime()}
		if status.GetAlive() {
			r.delay = status.GetDelay()
			if r.delay <= 0 {
				r.delay = 1
			}
		} else {
			r.err = status.GetLastErrorReason()
			if r.err == "" {
				r.err = "outbound is not alive"
			}
		}
		result[status.GetOutboundTag()] = r
	}
	return result
}

func (c *xrayControl) selectTarget(balancer, target string) error {
	overrider, ok := c.router.(routing.BalancerOverrider)
	if !ok {
		return errors.New("xray: router does not support balancer override")
	}
	if target != "" {
		if !c.isCandidate(balancer, target) {
			return fmt.Errorf("xray: %q is not an outbound of balancer %q", target, balancer)
		}
	}
	return overrider.SetOverrideTarget(balancer, target)
}

func (c *xrayControl) isCandidate(balancer, target string) bool {
	if c.view.Routing == nil || c.ohm == nil || c.ohm.GetHandler(target) == nil {
		return false
	}
	for _, b := range c.view.Routing.Balancers {
		if b.Tag != balancer {
			continue
		}
		for _, candidate := range c.candidates(b.Selector) {
			if candidate == target {
				return true
			}
		}
	}
	return false
}

type xrayDelayResult struct {
	Delay int64  `json:"delay"`
	Error string `json:"error,omitempty"`
}

func (c *xrayControl) delay(tags []string, rawURL string, timeout time.Duration) (map[string]xrayDelayResult, error) {
	if c.dispatcher == nil || tagged.Dialer == nil {
		return nil, errors.New("xray: delay test is unavailable")
	}
	if rawURL == "" {
		rawURL = xrayDefaultProbeURL
	}
	if u, err := url.Parse(rawURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("xray: invalid test URL %q", rawURL)
	}
	if timeout <= 0 {
		timeout = xrayDefaultProbeTimeout
	}
	if timeout > xrayMaxProbeTimeout {
		timeout = xrayMaxProbeTimeout
	}

	results := make(map[string]xrayDelayResult, len(tags))
	var (
		resultsMu sync.Mutex
		wg        sync.WaitGroup
		slots     = make(chan struct{}, xrayProbeConcurrency)
	)
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, seen := results[tag]; seen {
			continue
		}
		results[tag] = xrayDelayResult{}
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			var r xrayDelayResult
			if c.ohm == nil || c.ohm.GetHandler(tag) == nil {
				r.Error = "outbound does not exist"
			} else if ms, err := c.probe(tag, rawURL, timeout); err != nil {
				r.Error = err.Error()
			} else {
				r.Delay = ms
			}

			c.probeMu.Lock()
			c.probes[tag] = xrayProbeResult{delay: r.Delay, testedAt: time.Now().Unix(), err: r.Error}
			c.probeMu.Unlock()

			resultsMu.Lock()
			results[tag] = r
			resultsMu.Unlock()
		}(tag)
	}
	wg.Wait()
	return results, nil
}

func (c *xrayControl) probe(tag, rawURL string, timeout time.Duration) (ms int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			ms, err = 0, fmt.Errorf("probe panicked: %v", r)
		}
	}()

	transport := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			dest, err := xnet.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}
			return tagged.Dialer(c.ctx, c.dispatcher, dest, tag)
		},
		TLSHandshakeTimeout: timeout,
		DisableKeepAlives:   true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	response, err := client.Do(request)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if urlErr.Timeout() {
				return 0, errors.New("timeout")
			}
			err = urlErr.Err
		}
		return 0, err
	}
	_ = response.Body.Close()
	ms = time.Since(start).Milliseconds()
	if ms <= 0 {
		ms = 1
	}
	return ms, nil
}

type xrayTraffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type xrayStats struct {
	Enabled    bool                   `json:"enabled"`
	Up         int64                  `json:"up"`
	Down       int64                  `json:"down"`
	Outbounds  map[string]xrayTraffic `json:"outbounds"`
	Inbounds   map[string]xrayTraffic `json:"inbounds"`
	Goroutines int                    `json:"goroutines"`
	Uptime     int64                  `json:"uptime"`
}

type xrayCounterVisitor interface {
	VisitCounters(func(string, featurestats.Counter) bool)
}

func (c *xrayControl) counterVisitor() xrayCounterVisitor {
	if c.stats == nil {
		return nil
	}
	if _, noop := c.stats.(featurestats.NoopManager); noop {
		return nil
	}
	if _, noop := c.stats.(*featurestats.NoopManager); noop {
		return nil
	}
	visitor, _ := c.stats.(xrayCounterVisitor)
	return visitor
}

func (c *xrayControl) statsSnapshot() xrayStats {
	snapshot := xrayStats{
		Outbounds:  map[string]xrayTraffic{},
		Inbounds:   map[string]xrayTraffic{},
		Goroutines: runtime.NumGoroutine(),
		Uptime:     int64(time.Since(c.started).Seconds()),
	}
	visitor := c.counterVisitor()
	if visitor == nil {
		return snapshot
	}
	snapshot.Enabled = true
	visitor.VisitCounters(func(name string, counter featurestats.Counter) bool {
		parts := strings.Split(name, ">>>")
		if len(parts) != 4 || parts[2] != "traffic" {
			return true
		}
		var bucket map[string]xrayTraffic
		switch parts[0] {
		case "outbound":
			bucket = snapshot.Outbounds
		case "inbound":
			bucket = snapshot.Inbounds
		default:
			return true
		}
		value := counter.Value()
		traffic := bucket[parts[1]]
		switch parts[3] {
		case "uplink":
			traffic.Up = value
		case "downlink":
			traffic.Down = value
		default:
			return true
		}
		bucket[parts[1]] = traffic
		return true
	})
	for _, traffic := range snapshot.Outbounds {
		snapshot.Up += traffic.Up
		snapshot.Down += traffic.Down
	}
	return snapshot
}

type xrayLogEntry struct {
	Seq   int64  `json:"seq"`
	Time  int64  `json:"time"`
	Level string `json:"level"`
	Kind  string `json:"kind"`
	Text  string `json:"text"`
}

type xrayLogPage struct {
	Entries []xrayLogEntry `json:"entries"`
	Next    int64          `json:"next"`
}

type xrayLogBuffer struct {
	lastPoll    atomic.Int64
	maxSeverity atomic.Int32

	mu      sync.Mutex
	entries []xrayLogEntry
	head    int
	seq     int64
}

func newXrayLogBuffer(capacity int) *xrayLogBuffer {
	b := &xrayLogBuffer{entries: make([]xrayLogEntry, 0, capacity)}
	b.maxSeverity.Store(int32(commonlog.Severity_Info))
	return b
}

func (b *xrayLogBuffer) active() bool {
	last := b.lastPoll.Load()
	return last != 0 && time.Since(time.Unix(0, last)) < xrayLogIdleTimeout
}

func (b *xrayLogBuffer) capture(msg commonlog.Message) {
	if !b.active() {
		return
	}
	entry := xrayLogEntry{Time: time.Now().UnixMilli()}
	switch m := msg.(type) {
	case *commonlog.GeneralMessage:
		if int32(m.Severity) > b.maxSeverity.Load() {
			return
		}
		entry.Kind = "general"
		entry.Level = xraySeverityName(m.Severity)
		entry.Text = fmt.Sprint(m.Content)
	case *commonlog.AccessMessage:
		entry.Kind, entry.Level, entry.Text = "access", "info", m.String()
	case *commonlog.DNSLog:
		entry.Kind, entry.Level, entry.Text = "dns", "info", m.String()
	default:
		return
	}

	b.mu.Lock()
	b.seq++
	entry.Seq = b.seq
	if len(b.entries) < cap(b.entries) {
		b.entries = append(b.entries, entry)
	} else {
		b.entries[b.head] = entry
		b.head = (b.head + 1) % len(b.entries)
	}
	b.mu.Unlock()
}

func (b *xrayLogBuffer) read(after int64, limit int, level string) xrayLogPage {
	b.lastPoll.Store(time.Now().UnixNano())
	switch strings.ToLower(level) {
	case "debug":
		b.maxSeverity.Store(int32(commonlog.Severity_Debug))
	case "warning", "warn":
		b.maxSeverity.Store(int32(commonlog.Severity_Warning))
	case "error":
		b.maxSeverity.Store(int32(commonlog.Severity_Error))
	case "info":
		b.maxSeverity.Store(int32(commonlog.Severity_Info))
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	page := xrayLogPage{Entries: []xrayLogEntry{}, Next: b.seq}
	if after > b.seq {
		after = 0
	}
	for i := 0; i < len(b.entries); i++ {
		entry := b.entries[(b.head+i)%len(b.entries)]
		if entry.Seq > after {
			page.Entries = append(page.Entries, entry)
		}
	}
	if limit > 0 && len(page.Entries) > limit {
		page.Entries = page.Entries[len(page.Entries)-limit:]
	}
	return page
}

func xraySeverityName(s commonlog.Severity) string {
	switch s {
	case commonlog.Severity_Error:
		return "error"
	case commonlog.Severity_Warning:
		return "warning"
	case commonlog.Severity_Debug:
		return "debug"
	default:
		return "info"
	}
}

type xrayLogTee struct {
	next commonlog.Handler
	buf  *xrayLogBuffer
}

func (t *xrayLogTee) Handle(msg commonlog.Message) {
	if t.next != nil {
		t.next.Handle(msg)
	}
	t.buf.capture(msg)
}
