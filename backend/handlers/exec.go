package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
)

type ExecHandler struct {
	docker *dockerclient.Client
	logger *slog.Logger
}

func NewExecHandler(logger *slog.Logger) (*ExecHandler, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	return &ExecHandler{docker: cli, logger: logger}, nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ExecResizeMessage struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
}

func (h *ExecHandler) HandleExec(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		http.Error(w, "invalid org id", http.StatusBadRequest)
		return
	}

	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		http.Error(w, "invalid connection id", http.StatusBadRequest)
		return
	}

	containerID := chi.URLParam(r, "containerID")
	if containerID == "" {
		http.Error(w, "container id required", http.StatusBadRequest)
		return
	}

	_ = orgID
	_ = connID

	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("upgrading websocket", "error", err)
		return
	}
	defer conn.Close()

	h.logger.Info("exec session started",
		"org_id", orgID,
		"connection_id", connID,
		"container_id", containerID,
	)

	// First check if container exists and is running
	inspect, err := h.docker.ContainerInspect(r.Context(), containerID)
	if err != nil {
		h.logger.Error("container not found", "container_id", containerID, "error", err)
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mError: Container not found or not accessible\x1b[0m\r\n"))
		conn.WriteMessage(websocket.TextMessage, []byte("The container may have been removed or recreated. Please refresh the page.\r\n"))
		return
	}

	if !inspect.State.Running {
		h.logger.Error("container not running", "container_id", containerID, "state", inspect.State.Status)
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mError: Container is not running\x1b[0m\r\n"))
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Container state: %s\r\n", inspect.State.Status)))
		return
	}

	// Try different shells
	shells := []string{"/bin/sh", "/bin/bash", "/bin/ash"}
	var execID string
	var hijacked *types.HijackedResponse

	for _, shell := range shells {
		execResp, err := h.docker.ContainerExecCreate(r.Context(), containerID, container.ExecOptions{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			Cmd:          []string{shell},
		})
		if err != nil {
			h.logger.Debug("exec create failed", "shell", shell, "error", err)
			continue
		}

		execID = execResp.ID

		hijackedResp, err := h.docker.ContainerExecAttach(r.Context(), execID, container.ExecAttachOptions{
			Tty: true,
		})
		if err != nil {
			h.logger.Error("exec attach failed", "error", err)
			continue
		}

		hijacked = &hijackedResp
		break
	}

	if hijacked == nil {
		h.logger.Error("failed to exec into container", "container_id", containerID)
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mError: Failed to start shell in container\x1b[0m\r\n"))
		conn.WriteMessage(websocket.TextMessage, []byte("The container may not have a shell available.\r\n"))
		return
	}
	defer hijacked.Close()

	// Send initial welcome message
	conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[36m[ContainerScope Terminal]\x1b[0m\r\n"))
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\x1b[32mConnected to container: %s\x1b[0m\r\n\r\n", containerID[:12])))

	// Pipe: Docker stdout -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := hijacked.Reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					h.logger.Debug("docker read error", "error", err)
				}
				return
			}
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
		}
	}()

	// Pipe: WebSocket -> Docker stdin
	go func() {
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					h.logger.Debug("websocket read error", "error", err)
				}
				return
			}

			if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
				// Check if it's a resize message
				if len(msg) > 0 && msg[0] == '{' {
					var resizeMsg ExecResizeMessage
					if json.Unmarshal(msg, &resizeMsg) == nil && resizeMsg.Type == "resize" {
						h.docker.ContainerExecResize(r.Context(), execID, container.ResizeOptions{
							Width:  resizeMsg.Cols,
							Height: resizeMsg.Rows,
						})
						continue
					}
				}

				// Send to container stdin
				_, err := hijacked.Conn.Write(msg)
				if err != nil {
					h.logger.Debug("docker write error", "error", err)
					return
				}
			}
		}
	}()

	// Keep alive until disconnect
	for {
		time.Sleep(time.Second)
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}
