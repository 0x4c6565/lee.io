package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/0x4c6565/lee.io/pkg/tool"
	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type ServerOptions struct {
	Initialise bool
}

type Server struct {
	tools []tool.Tool
	opts  ServerOptions
}

func NewServer(opts ServerOptions) *Server {
	return &Server{opts: opts}
}

func (s *Server) WithTools(tools ...tool.Tool) *Server {
	s.tools = append(s.tools, tools...)
	return s
}

func (s *Server) Start(ctx context.Context) error {
	log.Info().Msg("Starting server")

	r := mux.NewRouter()
	c := cron.New()
	for _, t := range s.tools {
		for _, path := range t.Paths() {
			log.Trace().
				Str("method", t.Method()).
				Str("path", path).
				Msg("Adding route")
			r.HandleFunc(path, newHandler(t).handle)
		}

		if v, ok := t.(tool.ToolCron); ok {
			spec := v.Cron()
			if s.opts.Initialise {
				go spec.Func()
			}

			c.AddFunc(spec.Cron, spec.Func)
		}
	}

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/lee.io")))

	go c.Run()
	server := &http.Server{Addr: ":8080", Handler: r}

	var err error
	go func() {
		err = server.ListenAndServe()
	}()

	log.Info().Msg("Server started")

	<-ctx.Done()
	log.Debug().Msg("Server shutting down..")
	server.Shutdown(context.Background())
	log.Debug().Msg("Server shut down complete")

	log.Debug().Msg("Housekeeper shutting down..")
	cronCtx := c.Stop()
	<-cronCtx.Done()
	log.Debug().Msg("Housekeeper shutdown complete")

	return err
}

type handler struct {
	tool tool.Tool
}

func newHandler(tool tool.Tool) *handler {
	return &handler{tool: tool}
}

func (h *handler) handle(rw http.ResponseWriter, r *http.Request) {
	log.Trace().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Msgf("Handling request")
	accept := r.Header.Values("Accept")
	userAgent := r.Header.Values("User-agent")

	response, err := h.tool.Handle(r)

	acceptHeader := ""
	if len(accept) > 0 {
		acceptHeader = accept[0]
	}

	switch acceptHeader {
	case "application/json":
		if err != nil {
			jsonResponse(rw, 500, jsonErrorResponse{Error: err.Error()})
		} else {
			jsonResponse(rw, 200, response.Data)
		}
	default:
		if err != nil {
			plainResponse(rw, 500, err.Error())
		} else {
			plainResponse(rw, 200, response.Data.String())
		}
	}

	for _, h := range userAgent {
		if strings.Contains(h, "curl") {
			rw.Write([]byte("\n"))
			break
		}
	}
}

type jsonErrorResponse struct {
	Error string `json:"error"`
}

func jsonResponse(rw http.ResponseWriter, code int, data interface{}) {
	rw.Header().Add("Content-type", "application/json")
	rw.WriteHeader(code)
	j, _ := json.Marshal(data)
	rw.Write(j)
}

func plainResponse(rw http.ResponseWriter, code int, data string) {
	rw.Header().Add("Content-type", "text/plain")
	rw.WriteHeader(code)
	rw.Write([]byte(data))
}
