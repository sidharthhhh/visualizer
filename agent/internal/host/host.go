package host

import (
	"os"
	"runtime"
)

type Info struct {
	Hostname     string
	OS           string
	Kernel       string
	CPUCores     int32
	MemTotal     int64
	AgentVersion string
}

func Collect(version string) (*Info, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return &Info{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Kernel:       runtime.GOARCH,
		CPUCores:     int32(runtime.NumCPU()),
		MemTotal:     0,
		AgentVersion: version,
	}, nil
}
