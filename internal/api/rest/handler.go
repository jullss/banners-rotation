package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jullss/banners-rotation/internal/service"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /slots/{slot_id}/banners/{banner_id}", h.addBannerToSlot)
	mux.HandleFunc("DELETE /slots/{slot_id}/banners/{banner_id}", h.removeBannerFromSlot)
	mux.HandleFunc("GET /slots/{slot_id}/choose", h.chooseBanner)
	mux.HandleFunc("POST /slots/{slot_id}/banners/{banner_id}/click", h.recordClick)
}

func (h *Handler) addBannerToSlot(w http.ResponseWriter, r *http.Request) {
	slotID, bannerID, err := slotAndBanner(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.svc.AddBannerToSlot(r.Context(), slotID, bannerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeBannerFromSlot(w http.ResponseWriter, r *http.Request) {
	slotID, bannerID, err := slotAndBanner(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.svc.RemoveBannerFromSlot(r.Context(), slotID, bannerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) chooseBanner(w http.ResponseWriter, r *http.Request) {
	slotID, err := pathInt(r, "slot_id")
	if err != nil {
		http.Error(w, "invalid slot_id", http.StatusBadRequest)
		return
	}
	groupID, err := queryInt(r, "group_id")
	if err != nil {
		http.Error(w, "invalid group_id", http.StatusBadRequest)
		return
	}
	banner, err := h.svc.ChooseBanner(r.Context(), slotID, groupID)
	if err != nil {
		if errors.Is(err, errNoContent) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(banner) //nolint:errcheck
}

func (h *Handler) recordClick(w http.ResponseWriter, r *http.Request) {
	slotID, bannerID, err := slotAndBanner(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	groupID, err := queryInt(r, "group_id")
	if err != nil {
		http.Error(w, "invalid group_id", http.StatusBadRequest)
		return
	}
	if err := h.svc.RecordClick(r.Context(), slotID, bannerID, groupID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var errNoContent = errors.New("no banners in slot")

func slotAndBanner(r *http.Request) (int64, int64, error) {
	slotID, err := pathInt(r, "slot_id")
	if err != nil {
		return 0, 0, errors.New("invalid slot_id")
	}
	bannerID, err := pathInt(r, "banner_id")
	if err != nil {
		return 0, 0, errors.New("invalid banner_id")
	}
	return slotID, bannerID, nil
}

func pathInt(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func queryInt(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
}
