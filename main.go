/*
This file is part of GigaPaste.

GigaPaste is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

GigaPaste is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with GigaPaste. If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"app/internal/dbo"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed data/settings.json
var settingsFile string

// theoretically won't ever conflict with generated url, because generated url won't contain dot "."
func serveFile(w http.ResponseWriter, r *http.Request, next func(w2 http.ResponseWriter, r2 *http.Request)) {

	// Remove leading slash from the URL path
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "" {
		http.Redirect(w, r, "/index.html", http.StatusFound)
		return
	}

	// Open the file from the embedded file system
	file, err := staticFiles.ReadFile("static/" + path)
	if err != nil {
		//if GET url is not found in static files, then we pass the request to the next handler (file download handler)
		next(w, r)
		return
	}

	// Detect the MIME type based on file extension
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set headers and write the file to the response
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(file)))
	w.WriteHeader(http.StatusOK)
	w.Write(file)
}

func main() {
	bindAddress := flag.String("bind", ":80", "HTTP binding address")
	flag.Parse()

	if _, err := os.Stat("./uploads/"); os.IsNotExist(err) {
		err := os.MkdirAll("./uploads", os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}

	if _, err := os.Stat("./data/"); os.IsNotExist(err) {
		err := os.MkdirAll("./data", os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}

		err = os.WriteFile("./data/settings.json", []byte(settingsFile), 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

	}

	db, err := dbo.Dial(context.Background(), "file:./data/database.db?cache=shared")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	queries := dbo.New(db)

	InitSettings()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodGet {

			serveFile(w, r, func(w2 http.ResponseWriter, r2 *http.Request) {

				DownloadHandler(w2, r2, queries)

			})

		}

		if r.Method == http.MethodPost {

			r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*Global.FileSizeLimit)
			FileHandler(w, r, queries)

			//before the multipart form is parsed, it will be written to temporary folder, make sure to clean it after we done
			if r.MultipartForm != nil {
				err := r.MultipartForm.RemoveAll()
				if err != nil {
					fmt.Println(err)
				}

			}

		}

	})

	http.HandleFunc("/postText", func(w http.ResponseWriter, r *http.Request) {

		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*Global.TextSizeLimit)
		TextHandler(w, r, queries)

	})

	go CheckExpiration(queries)

	server := &http.Server{Addr: *bindAddress}
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("Shutting down")
		if err := server.Close(); err != nil {
			fmt.Println(err)
		}
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	log.Println("Server running", *bindAddress)
	log.Fatal(server.ListenAndServe())
}
