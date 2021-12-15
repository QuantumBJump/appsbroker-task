package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	_ "github.com/go-sql-driver/mysql"
)

const defaultAddr = ":8080"

type templateData struct {
	Message      string
	CloudStorage string
	CloudSQL     []SQLObj
}

type SQLObj struct {
	ID  int    `json:"id"`
	Foo string `json:"Foo"`
	Bar string `json:"Bar"`
	Baz string `json:"Baz"`
}

var (
	data templateData
	tmpl *template.Template
)

func main() {
	println("Hello world!")

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
	contents := new(strings.Builder)
	_, err = io.Copy(contents, r)
	if err != nil {
		log.Fatalf("Failed to read contents: %+v", err)
	}

	content_string := contents.String()
	r.Close()

	log.Printf("File contents: %s", content_string)

	// SQL interaction
	dbUsername := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbConnect := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", dbUsername, dbPassword, dbName)

	db, err := sql.Open("mysql", dbConnect)
	if err != nil {
		log.Fatalf("Unable to connect to database: %+v", err)
	}

	results, err := db.Query("SELECT * FROM testdata")
	if err != nil {
		log.Fatalf("Failed to get results: %+v", err)
	}

	objects := []SQLObj{}

	for results.Next() {
		var object SQLObj
		err = results.Scan(&object.ID, &object.Foo, &object.Bar, &object.Baz)
		if err != nil {
			log.Fatalf("Failed to scan results: %+v", err)
		}
		objects = append(objects, object)
		log.Printf("ID: %d, Foo: %s, Bar: %s, Baz: %s", object.ID, object.Foo, object.Bar, object.Baz)
	}

	db.Close()
	// Generate and display template
	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %+v", err)
	}

	tmpl = t

	data = templateData{
		Message:      "Hello, world!",
		CloudStorage: content_string,
		CloudSQL:     objects,
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
