package flows

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Protocol string

const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolTCP6 Protocol = "tcp6"
	ProtocolUDP  Protocol = "udp"
	ProtocolUDP6 Protocol = "udp6"
)

type SocketState int

const (
	StateEstablished SocketState = 1
	StateSynSent     SocketState = 2
	StateSynRecv     SocketState = 3
	StateFinWait1    SocketState = 4
	StateFinWait2    SocketState = 5
	StateTimeWait    SocketState = 6
	StateClose       SocketState = 7
	StateCloseWait   SocketState = 8
	StateLastAck     SocketState = 9
	StateListen      SocketState = 10
)

type Connection struct {
	Protocol    Protocol
	LocalIP     string
	LocalPort   uint16
	RemoteIP    string
	RemotePort  uint16
	State       SocketState
	Inode       string
	PID         int
	ContainerID string
}

type Edge struct {
	SrcContainerID string
	DstContainerID string
	DstIP          string
	DstPort        uint16
	Protocol       Protocol
}

func ParseProcNet(proto Protocol) ([]Connection, error) {
	var path string
	switch proto {
	case ProtocolTCP:
		path = "/proc/net/tcp"
	case ProtocolTCP6:
		path = "/proc/net/tcp6"
	case ProtocolUDP:
		path = "/proc/net/udp"
	case ProtocolUDP6:
		path = "/proc/net/udp6"
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer file.Close()

	var conns []Connection
	scanner := bufio.NewScanner(file)
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		localAddr := fields[1]
		remoteAddr := fields[2]
		stateHex := fields[3]
		inode := fields[9]

		localIP, localPort, err := parseAddress(localAddr)
		if err != nil {
			continue
		}

		remoteIP, remotePort, err := parseAddress(remoteAddr)
		if err != nil {
			continue
		}

		stateInt, err := strconv.ParseInt(stateHex, 16, 32)
		if err != nil {
			continue
		}

		conns = append(conns, Connection{
			Protocol:   proto,
			LocalIP:    localIP,
			LocalPort:  localPort,
			RemoteIP:   remoteIP,
			RemotePort: remotePort,
			State:      SocketState(stateInt),
			Inode:      inode,
		})
	}

	return conns, nil
}

func parseAddress(addr string) (string, uint16, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s", addr)
	}

	ipHex := parts[0]
	portHex := parts[1]

	ip, err := parseHexIP(ipHex)
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return "", 0, err
	}

	return ip, uint16(port), nil
}

func parseHexIP(hex string) (string, error) {
	if len(hex) == 8 {
		var parts []string
		for i := 6; i >= 0; i -= 2 {
			b, err := strconv.ParseUint(hex[i:i+2], 16, 8)
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("%d", b))
		}
		return strings.Join(parts, "."), nil
	}

	if len(hex) == 32 {
		var parts []string
		for i := 0; i < 32; i += 4 {
			group := hex[i : i+4]
			b, err := strconv.ParseUint(group, 16, 16)
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("%x", b))
		}
		return strings.Join(parts, ":"), nil
	}

	return "", fmt.Errorf("invalid IP hex length: %d", len(hex))
}

func MapSocketsToContainers(conns []Connection, pidToContainer map[int]string) []Connection {
	for i := range conns {
		if conns[i].PID > 0 {
			if containerID, ok := pidToContainer[conns[i].PID]; ok {
				conns[i].ContainerID = containerID
			}
		}
	}
	return conns
}

func DeriveEdges(conns []Connection) []Edge {
	edgeMap := make(map[string]Edge)

	for _, conn := range conns {
		if conn.State != StateEstablished {
			continue
		}

		if conn.ContainerID == "" {
			continue
		}

		if conn.RemotePort == 0 {
			continue
		}

		key := fmt.Sprintf("%s:%s:%d:%s", conn.ContainerID, conn.RemoteIP, conn.RemotePort, conn.Protocol)

		dstContainerID := ""
		for _, other := range conns {
			if other.ContainerID != "" && other.LocalPort == conn.RemotePort {
				dstContainerID = other.ContainerID
				break
			}
		}

		edgeMap[key] = Edge{
			SrcContainerID: conn.ContainerID,
			DstContainerID: dstContainerID,
			DstIP:          conn.RemoteIP,
			DstPort:        conn.RemotePort,
			Protocol:       conn.Protocol,
		}
	}

	edges := make([]Edge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, edge)
	}
	return edges
}

func GetContainerPIDs() (map[int]string, error) {
	result := make(map[int]string)

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		cgroupPath := filepath.Join("/proc", entry.Name(), "cgroup")
		cgroupData, err := os.ReadFile(cgroupPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(cgroupData), "\n")
		for _, line := range lines {
			if strings.Contains(line, "docker") || strings.Contains(line, "containerd") {
				parts := strings.Split(line, "/")
				for _, part := range parts {
					if len(part) == 64 {
						result[pid] = part[:12]
						break
					}
				}
			}
		}
	}

	return result, nil
}
