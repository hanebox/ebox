package system

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"

	"ebox/app/support/version"
)

type DockerInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Info struct {
	Hostname        string     `json:"hostname"`
	Platform        string     `json:"platform"`
	PlatformVersion string     `json:"platform_version"`
	KernelVersion   string     `json:"kernel_version"`
	Arch            string     `json:"arch"`
	CPUs            int        `json:"cpus"`
	MemTotal        uint64     `json:"mem_total"`
	Uptime          uint64     `json:"uptime"`
	GoVersion       string     `json:"go_version"`
	PanelVersion    string     `json:"panel_version"`
	PanelCommit     string     `json:"panel_commit"`
	Docker          DockerInfo `json:"docker"`
}

func Get(dataDir string) Info {
	info := Info{
		Arch:         runtime.GOARCH,
		CPUs:         runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		PanelVersion: version.Version,
		PanelCommit:  version.Commit,
		Docker:       dockerInfo(),
	}

	if hostInfo, err := host.Info(); err == nil {
		info.Hostname = hostInfo.Hostname
		info.Platform = hostInfo.Platform
		info.PlatformVersion = hostInfo.PlatformVersion
		info.KernelVersion = hostInfo.KernelVersion
		info.Uptime = hostInfo.Uptime
	} else if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	if virtual, err := mem.VirtualMemory(); err == nil {
		info.MemTotal = virtual.Total
	}

	_ = dataDir

	return info
}

var (
	dockerMu        sync.Mutex
	dockerCached    DockerInfo
	dockerCheckedAt time.Time
)

func dockerInfo() DockerInfo {
	dockerMu.Lock()
	defer dockerMu.Unlock()

	if time.Since(dockerCheckedAt) < 30*time.Second {
		return dockerCached
	}

	dockerCached = probeDocker()
	dockerCheckedAt = time.Now()

	return dockerCached
}

func probeDocker() DockerInfo {
	binary, err := exec.LookPath("docker")
	if err != nil {
		return DockerInfo{Available: false, Error: "docker binary not found"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, binary, "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		if len(message) > 300 {
			message = message[:300]
		}

		return DockerInfo{Available: false, Error: message}
	}

	return DockerInfo{Available: true, Version: strings.TrimSpace(string(out))}
}
