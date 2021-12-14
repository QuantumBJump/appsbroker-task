package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	storage "cloud.google.com/go/storage"
)

const defaultAddr = ":8080"

type templateData struct {
	Message      string
	CloudStorage string
}

var (
	data templateData
	tmpl *template.Template
)

func main() {
	// Create Google Storage client
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client!")
	}

	bkt := client.Bucket("appsbroker-task-static-blobstore")

	obj := bkt.Object("testdata.txt")
	r, err := obj.NewReader(ctx)
	if err != nil {
		log.Fatalf("Failed to create object reader: %+v", err)
	}
	defer r.Close()
	contents := new(strings.Builder)
	_, err = io.Copy(contents, r)
	if err != nil {
		log.Fatalf("Failed to read contents: %+v", err)
	}

	content_string := contents.String()

	log.Printf("File contents: %s", content_string)

	println("Hello world!")
	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %+v", err)
	}

	tmpl = t

	data = templateData{
		Message:      "Hello, world!",
		CloudStorage: content_string,
	}

	addr := defaultAddr
	// $PORT environment variable is provided in the k8s deployment
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("Server listening on port %s", addr)

	http.HandleFunc("/", home)

	fs := http.FileServer(http.Dir("template/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server listening error: %+v", err)
	}
}

// home responds to requests by rendering an html page
func home(w http.ResponseWriter, r *http.Request) {
	log.Printf("Hello world! Received request: %s %s", r.Method, r.URL.Path)
	if err := tmpl.Execute(w, data); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Printf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}

}
