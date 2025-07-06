package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pattern_report.html")
	})
	
	fmt.Println("Server starting at http://localhost:8080")
	fmt.Println("Open http://localhost:8080 in your browser to view the report")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}