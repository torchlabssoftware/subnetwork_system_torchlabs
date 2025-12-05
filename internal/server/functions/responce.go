package server

import (
	"encoding/json"
	"log"
	"net/http"
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
	//if code > 499 {
	log.Println("Responding with 5xx error:", message, err)
	//}

	payload := struct {
		Error string `json:"error"`
	}{Error: message}

	RespondwithJSON(w, code, payload)
}
