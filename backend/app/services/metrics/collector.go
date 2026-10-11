package metrics

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"

	"ebox/app/facades"
)

const (
	sampleInterval     = 3 * time.Second
	shortWindowSamples = 400
	longWindowSamples  = 1440
)

type Sample struct {
	At          time.Time `json:"at"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemUsed     uint64    `json:"mem_used"`
	MemTotal    uint64    `json:"mem_total"`
	MemPercent  float64   `json:"mem_percent"`
	SwapUsed    uint64    `json:"swap_used"`
	SwapTotal   uint64    `json:"swap_total"`
	Load1       float64   `json:"load1"`
	Load5       float64   `json:"load5"`
	Load15      float64   `json:"load15"`
	DiskUsed    uint64    `json:"disk_used"`
	DiskTotal   uint64    `json:"disk_total"`
	DiskPercent float64   `json:"disk_percent"`
	NetRxRate   float64   `json:"net_rx_rate"`
	NetTxRate   float64   `json:"net_tx_rate"`
	Uptime      uint64    `json:"uptime"`
}

type Collector struct {
	mu       sync.RWMutex
	short    []Sample
	long     []Sample
	last     *Sample
	diskPath string

	lastNetRx uint64
	lastNetTx uint64
	lastNetAt time.Time

	stop     chan struct{}
	stopOnce sync.Once
}

var defaultCollector = NewCollector("/")

func NewCollector(diskPath string) *Collector {
	return &Collector{
		short:    make([]Sample, 0, shortWindowSamples),
		long:     make([]Sample, 0, longWindowSamples),
		diskPath: diskPath,
		stop:     make(chan struct{}),
	}
}

func Default() *Collector {
	return defaultCollector
}

func Init(diskPath string) *Collector {
	defaultCollector = NewCollector(diskPath)

	return defaultCollector
}

func (c *Collector) Signature() string {
	return "ebox:metrics-collector"
}

func (c *Collector) ShouldRun() bool {
	return true
}

func (c *Collector) Run() error {
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	c.collectAndStore()

	for {
		select {
		case <-c.stop:
			return nil
		case <-ticker.C:
			c.collectAndStore()
		}
	}
}

func (c *Collector) Shutdown() error {
	c.stopOnce.Do(func() { close(c.stop) })

	return nil
}

func (c *Collector) Current() (Sample, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.last == nil {
		return Sample{}, false
	}

	return *c.last, true
}

func (c *Collector) History(window string) []Sample {
	c.mu.RLock()
	defer c.mu.RUnlock()

	source := c.short
	if window == "24h" {
		source = c.long
	}

	out := make([]Sample, len(source))
	copy(out, source)

	return out
}

func (c *Collector) collectAndStore() {
	sample, err := c.collect()
	if err != nil {
		facades.Log().Warning("metrics: collect failed", "error", err)
		return
	}

	c.mu.Lock()
	c.last = &sample
	c.short = appendRing(c.short, sample, shortWindowSamples)

	if len(c.long) == 0 || sample.At.Sub(c.long[len(c.long)-1].At) >= time.Minute {
		c.long = appendRing(c.long, sample, longWindowSamples)
	}
	c.mu.Unlock()
}

func (c *Collector) collect() (Sample, error) {
	now := time.Now().UTC()
	sample := Sample{At: now}

	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		sample.CPUPercent = cpuPercents[0]
	}

	if virtual, err := mem.VirtualMemory(); err == nil {
		sample.MemUsed = virtual.Used
		sample.MemTotal = virtual.Total
		sample.MemPercent = virtual.UsedPercent
	}

	if swap, err := mem.SwapMemory(); err == nil {
		sample.SwapUsed = swap.Used
		sample.SwapTotal = swap.Total
	}

	if avg, err := load.Avg(); err == nil {
		sample.Load1 = avg.Load1
		sample.Load5 = avg.Load5
		sample.Load15 = avg.Load15
	}

	if usage, err := disk.Usage(c.diskPath); err == nil {
		sample.DiskUsed = usage.Used
		sample.DiskTotal = usage.Total
		sample.DiskPercent = usage.UsedPercent
	}

	if counters, err := gopsnet.IOCounters(false); err == nil && len(counters) > 0 {
		totalRx := counters[0].BytesRecv
		totalTx := counters[0].BytesSent

		if !c.lastNetAt.IsZero() {
			elapsed := now.Sub(c.lastNetAt).Seconds()
			if elapsed > 0 {
				sample.NetRxRate = float64(totalRx-c.lastNetRx) / elapsed
				sample.NetTxRate = float64(totalTx-c.lastNetTx) / elapsed
			}
		}

		c.lastNetRx = totalRx
		c.lastNetTx = totalTx
		c.lastNetAt = now
	}

	if uptime, err := host.Uptime(); err == nil {
		sample.Uptime = uptime
	}

	return sample, nil
}

func appendRing[T any](ring []T, value T, capacity int) []T {
	if len(ring) < capacity {
		return append(ring, value)
	}

	copy(ring, ring[1:])
	ring[len(ring)-1] = value

	return ring
}
