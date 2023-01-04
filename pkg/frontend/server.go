package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/radekg/boos/configs"
)

// Server is a basic implementation of a frontend server.
type Server struct {
	backEndConfig *configs.BackEndConfig
	logger        hclog.Logger
}

// ServeListen creates a new frontend server and attempts to listen.
func ServeListen(backEndConfig *configs.BackEndConfig, frontEndConfig *configs.FrontEndConfig, logger hclog.Logger) error {

	srv := Server{
		backEndConfig: backEndConfig,
		logger:        logger,
	}

	fs := http.FileServer(http.Dir("./public"))
	http.HandleFunc("/backend", srv.backendHandler)
	http.Handle("/", fs)

	chanErr := make(chan error, 1)

	go func() {
		err := http.ListenAndServe(frontEndConfig.BindAddress, nil)
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
	Address string `json:"address" mapstructure:"address"`
}

func (s *Server) backendHandler(w http.ResponseWriter, r *http.Request) {
	response := &BackendResponse{
		Address: s.backEndConfig.ExternalAddress,
	}
	raw, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed marshaling response: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", "application/json")
	w.Write(raw)
}
