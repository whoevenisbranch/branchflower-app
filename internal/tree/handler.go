package tree

import (
	"net/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(service *Service) Handler {
	return Handler{
		svc: service,
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	// h.svc.GetUser(ctx)
	w.Write([]byte("this is tree home"))
}
