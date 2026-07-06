package main

import (
	"encoding/json"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"message": "Your Go API is working!",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/api/hello", helloHandler)

	println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}