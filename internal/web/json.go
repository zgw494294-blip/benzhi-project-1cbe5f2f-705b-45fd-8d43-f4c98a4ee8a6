package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"tape-preservation-gate/internal/domain"
	"tape-preservation-gate/internal/workflow"
)

const maxBodyBytes = 1 << 20

type errorEnvelope struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Invalid("请求 JSON 无效：%v", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.Invalid("请求体只能包含一个 JSON 对象")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal"
	var de *domain.DomainError
	if errors.As(err, &de) {
		switch de.Code {
		case domain.ErrInvalid:
			status, code = http.StatusBadRequest, string(de.Code)
		case domain.ErrNotFound:
			status, code = http.StatusNotFound, string(de.Code)
		case domain.ErrState:
			status, code = http.StatusUnprocessableEntity, string(de.Code)
		default:
			code = string(de.Code)
		}
	} else if workflow.IsConflict(err) {
		status, code = http.StatusConflict, "version_conflict"
	}
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: err.Error()}})
}

func commandMeta(r *http.Request, expected uint64, actor string) workflow.CommandMeta {
	return workflow.CommandMeta{ExpectedVersion: expected, IdempotencyKey: r.Header.Get("Idempotency-Key"), Actor: actor}
}
