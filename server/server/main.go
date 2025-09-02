package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Payload struct {
	Content string `json:"content"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var p Payload
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&p); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	fmt.Println("Received document:\n", p.Content)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoint", handler)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	fmt.Println("Server listening on :8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("server error:", err)
	}
}
