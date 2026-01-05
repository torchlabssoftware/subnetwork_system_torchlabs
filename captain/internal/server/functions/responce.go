package server

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
)

func RespondwithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("faild to marchel json responce: %v/n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "Application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func RespondwithError(w http.ResponseWriter, code int, message string, err error) {
	_, file, line, ok := runtime.Caller(1)

	if ok {
		shortFile := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				shortFile = file[i+1:]
				break
			}
		}
		log.Printf("[ERROR] %s:%d : %v", shortFile, line, err)
	} else {
		log.Printf("[ERROR] %v", err)
	}

	payload := struct {
		Error string `json:"error"`
	}{Error: message}

	RespondwithJSON(w, code, payload)
}
