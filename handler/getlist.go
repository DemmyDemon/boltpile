package handler

import (
	"net/http"

	"github.com/DemmyDemon/boltpile/storage"
)

func GetList(eg storage.EntryGetter, config storage.Config, limiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Actually handle request
		SendMessage(w, http.StatusInternalServerError, NOT_IMPLEMENTED)
	}
}
