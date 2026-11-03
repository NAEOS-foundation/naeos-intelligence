package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
)

type requestIDKey struct{}
func contextWithRequestID(r *http.Request,id string) context.Context { return context.WithValue(r.Context(),requestIDKey{},id) }
func requestIDFrom(r *http.Request) string { id,_:=r.Context().Value(requestIDKey{}).(string); return id }

type problem struct {
	Type string `json:"type"`
	Title string `json:"title"`
	Status int `json:"status"`
	Code string `json:"code"`
	Detail string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func writeProblem(w http.ResponseWriter,r *http.Request,status int,code,detail string) {
	w.Header().Set("Content-Type","application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Type:"about:blank",Title:http.StatusText(status),Status:status,Code:code,Detail:detail,RequestID:requestIDFrom(r)})
}
func writeJSON(w http.ResponseWriter,status int,value any) {
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
