package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/radekg/boos/configs"
)

// Server is a basic implementation of a frontend server.
type Server struct {
	backEndConfig *configs.BackendConfig
	webRTCConfig  *configs.WebRTCConfig
	logger        hclog.Logger
}

// ServeListen creates a new frontend server and attempts to listen.
func ServeListen(backEndConfig *configs.BackendConfig,
	frontEndConfig *configs.FrontendConfig,
	webRTCConfig *configs.WebRTCConfig,
	logger hclog.Logger) error {

	srv := Server{
		backEndConfig: backEndConfig,
		logger:        logger,
		webRTCConfig:  webRTCConfig,
	}

	fileServer := http.FileServer(http.Dir(frontEndConfig.StaticDirectoryPath))

	chanErr := make(chan error, 1)

	go func() {
		err := http.ListenAndServe(frontEndConfig.BindAddress, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path == "/backend" {
				srv.backendHandler(w, r)
				return
			}

			path, err := filepath.Abs(r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			path = filepath.Join(frontEndConfig.StaticDirectoryPath, path)

			_, err = os.Stat(path)
			if os.IsNotExist(err) {
				// file does not exist, serve index.html
				http.ServeFile(w, r, filepath.Join(frontEndConfig.StaticDirectoryPath, frontEndConfig.StaticDirectoryRootDocument))
				return
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			fileServer.ServeHTTP(w, r)
		}))
		if err != nil {
			chanErr <- err
		}
	}()

	select {
	case err := <-chanErr:
		return err
	case <-time.After(time.Millisecond * 500):
		srv.logger.Info("Frontend server started and listening", "frontend-bind-address", frontEndConfig.BindAddress)
		// we are golden
	}
	return nil
}

// BackendResponse is the GET /backend response.
type BackendResponse struct {
	Address    string   `json:"address" mapstructure:"address"`
	ICEServers []string `json:"iceServers" mapstructure:"iceServers"`
}

func (s *Server) backendHandler(w http.ResponseWriter, r *http.Request) {
	response := &BackendResponse{
		Address:    s.backEndConfig.ExternalAddress,
		ICEServers: s.webRTCConfig.ICEServers,
	}
	raw, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed marshaling response: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", "application/json")
	w.Write(raw)
}
