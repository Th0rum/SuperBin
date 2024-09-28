/*
This file is part of GigaPaste.

GigaPaste is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

GigaPaste is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with GigaPaste. If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"crypto/cipher"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*
var templateFiles embed.FS

func FileHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	err := r.ParseMultipartForm(1024 * Global.StreamSizeLimit) //Limit memory usage (this is not limiting file size)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Retrieve data from the form
	var duration string
	var password string
	var burnStr string

	//we do these checks for curl support
	if len(r.MultipartForm.Value["duration"]) > 0 {
		duration = r.MultipartForm.Value["duration"][0]
	} else {
		duration = strconv.FormatInt(Global.CmdUploadDefaultDurationMinute, 10)
	}
	if len(r.MultipartForm.Value["pass"]) > 0 {
		password = r.MultipartForm.Value["pass"][0]
	} else {
		password = ""
	}
	if len(r.MultipartForm.Value["burn"]) > 0 {
		burnStr = r.MultipartForm.Value["burn"][0]
	} else {
		burnStr = ""
	}

	if burnStr == "" {
		burnStr = "false"
	}
	burn, err := strconv.ParseBool(burnStr)
	if err != nil {
		fmt.Println(err)
	}

	minutes, err := strconv.ParseInt(duration, 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	seconds := minutes * 60

	//if anyone manipulates the number to weird value
	if seconds <= 0 {
		return
	}

	//if over 200 years just set it to 200
	if seconds > 6311520000 {
		seconds = 6311520000
	}

	files := r.MultipartForm.File["file"]

	if len(files) == 0 {
		fmt.Println("file length == 0")
		return
	}

	passwordHash := ""
	passwordSalt := ""
	var encryptKey []byte = nil
	encryptSalt := ""

	if password != "" {

		salt, err := GenerateSalt()
		if err != nil {
			fmt.Println(err)
			return
		}

		passwordSalt = hex.EncodeToString(salt)
		passwordHash = hex.EncodeToString(GeneratePasswordHash(password, salt))

		salt2, err := GenerateSalt()
		if err != nil {
			fmt.Println(err)
			return
		}

		encryptSalt = hex.EncodeToString(salt2)
		encryptKey = GeneratePasswordHash(password, salt2)

	}

	//create unique file name + some random string as a protection in case there are 2 file uploads at the exact same time
	if len(files) == 1 {

		filePath := GenRandFileName("./uploads/", "")
		SingleFileWriter(files, filePath, encryptKey, func() {
			randUrl := GenRandPath(6, db)
			_, err = db.Exec("INSERT INTO data (id, type, fileName, filePath, burn, expire, passwordHash, passwordSalt, encryptSalt) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", randUrl, "file", files[0].Filename, filePath, burn, strconv.FormatInt(time.Now().Unix()+seconds, 10), passwordHash, passwordSalt, encryptSalt)
			if err != nil {
				fmt.Println(err)
				return
			}

			_, err = io.WriteString(w, r.Host+Global.BaseUrl+"/"+randUrl)
			if err != nil {
				fmt.Println(err)
				return
			}

		})

	}

	if len(files) >= 2 {

		filePath := GenRandFileName("./uploads/", ".zip")
		MultipleFileWriter(files, filePath, encryptKey, func() {
			randUrl := GenRandPath(6, db)
			_, err = db.Exec("INSERT INTO data (id, type, fileName, filePath, burn, expire, passwordHash, passwordSalt, encryptSalt) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", randUrl, "file", "files.zip", filePath, burn, strconv.FormatInt(time.Now().Unix()+seconds, 10), passwordHash, passwordSalt, encryptSalt)
			if err != nil {
				fmt.Println(err)
				return
			}

			_, err = io.WriteString(w, r.Host+Global.BaseUrl+"/"+randUrl)
			if err != nil {
				fmt.Println(err)
				return
			}

		})

	}

}

func TextHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	var jsonData struct {
		Duration int64  `json:"duration"`
		Text     string `json:"text"`
		Password string `json:"pass"`
		Burn     bool   `json:"burn"`
	}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&jsonData)
	if err != nil {
		fmt.Println(err)
		return
	}

	seconds := jsonData.Duration * 60

	if seconds <= 0 {
		return
	}

	//if over 200 years just set it to 200
	if seconds > 6311520000 {
		seconds = 6311520000
	}

	filePath := GenRandFileName("./uploads/", "")
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	password := jsonData.Password
	passwordHash := ""
	passwordSalt := ""
	var encryptKey []byte = nil
	encryptSalt := ""
	if password != "" {

		salt, err := GenerateSalt()
		if err != nil {
			fmt.Println(err)
			return
		}

		passwordSalt = hex.EncodeToString(salt)
		passwordHash = hex.EncodeToString(GeneratePasswordHash(password, salt))

		salt2, err := GenerateSalt()
		if err != nil {
			fmt.Println(err)
			return
		}

		encryptSalt = hex.EncodeToString(salt2)
		encryptKey = GeneratePasswordHash(password, salt2)

	}

	_, err = file.Write([]byte(jsonData.Text))
	if err != nil {
		fmt.Println(err)
		return
	}

	file.Close()

	randUrl := GenRandPath(6, db)
	_, err = db.Exec("INSERT INTO data (id, type, fileName, filePath, burn, expire, passwordHash, passwordSalt, encryptSalt) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", randUrl, "text", "", filePath, jsonData.Burn, strconv.FormatInt(time.Now().Unix()+seconds, 10), passwordHash, passwordSalt, encryptSalt)
	if err != nil {
		fmt.Println(err)
		return
	}

	if passwordHash != "" {

		err = EncryptFile(filePath, encryptKey)
		if err != nil {
			fmt.Println(err)
			return
		}

	}

	_, err = io.WriteString(w, r.Host+Global.BaseUrl+"/"+randUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

}

func DownloadHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	pathParts := strings.Split(r.URL.Path, "/")
	path := pathParts[len(pathParts)-1]

	var fType string
	var fFileName string
	var fFilePath string
	var fBurn string
	var fPasswordHash string
	var fPasswordSalt string
	var fEncryptSalt string

	decryptKey := r.URL.Query().Get("key")
	raw := r.URL.Query().Get("raw")

	err := db.QueryRow("SELECT type, fileName, filePath, burn, passwordHash, passwordSalt, encryptSalt FROM data WHERE id = ?", path).Scan(&fType, &fFileName, &fFilePath, &fBurn, &fPasswordHash, &fPasswordSalt, &fEncryptSalt)
	if err != nil {

		//path not found
		data, err := fs.ReadFile(templateFiles, "templates/notFound.html")
		if err != nil {
			fmt.Println(err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	//doesn't exist
	if fType == "" {
		return
	}

	//if link is password protected and no password is given
	if fPasswordHash != "" && decryptKey == "" {

		tmpl, err := template.ParseFS(templateFiles, "templates/authTemplate.html")
		if err != nil {
			fmt.Println(err)
			return
		}

		err = tmpl.Execute(w, struct{ Path string }{Path: Global.BaseUrl[1:] + "/" + path})

		if err != nil {
			fmt.Println(err)
			return
		}

		return

	}

	var decryptFileHash []byte
	if fPasswordHash != "" {

		passwordSalt, err := hex.DecodeString(fPasswordSalt)
		if err != nil {
			fmt.Println(err)
			return
		}

		decryptKeyHash := GeneratePasswordHash(decryptKey, passwordSalt)

		passwordHash, err := hex.DecodeString(fPasswordHash)
		if err != nil {
			fmt.Println(err)
			return
		}

		//if wrong password
		if !bytes.Equal(decryptKeyHash, passwordHash) {

			referer := r.Header.Get("Referer")
			if referer == "" {
				return
			}

			// Redirect to the referer URL
			http.Redirect(w, r, referer, http.StatusFound)
			return

		}

		encryptSalt, err := hex.DecodeString(fEncryptSalt)
		if err != nil {
			fmt.Println(err)
			return
		}

		decryptFileHash = GeneratePasswordHash(decryptKey, encryptSalt)

	}

	if fType == "file" {

		file, err := os.Open(fFilePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// Get the file stats to determine the size
		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Println(err)
			return
		}

		var (
			iv        []byte
			aesCTR    cipher.Stream
			nonceSize int
		)

		if fPasswordHash != "" {

			err, iv, aesCTR, nonceSize = GetDecryptInfo(fFilePath, decryptFileHash)
			if err != nil {

				fmt.Println(err)

			}

		}

		// Set the appropriate headers
		w.Header().Set("Content-Disposition", "attachment; filename="+fFileName)
		w.Header().Set("Content-Type", "application/octet-stream")
		if fPasswordHash != "" {
			w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size()-int64(nonceSize), 10))
		} else {
			w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
		}

		buffer := make([]byte, 1024*Global.StreamSizeLimit)

		//skip the IV
		if fPasswordHash != "" {
			file.Seek(int64(nonceSize), 0)
		}

		for {

			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Println(err)
				return
			}

			if n == 0 {

				if fBurn == "1" {

					file.Close()
					_, err = db.Exec("DELETE FROM data WHERE id = ?", path)
					if err != nil {
						fmt.Println(err)
						return
					}

					err = os.Remove(fFilePath)
					if err != nil {
						fmt.Println(err)
						return
					}

				}

				break
			}

			if fPasswordHash != "" {

				err, decrypted := DecryptFileStream(buffer[:n], n, iv, aesCTR)
				if err != nil {
					fmt.Println(err)
					return
				}

				if _, err := w.Write(decrypted); err != nil {
					fmt.Println(err)
					return
				}

			} else {

				if _, err := w.Write(buffer[:n]); err != nil {
					fmt.Println(err)
					return
				}

			}

			// Ensure that the client receives the data immediately
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			//need to add check for > 0 because Sleep(0) will just trigger context switch
			if Global.StreamThrottle > 0 {
				time.Sleep(time.Duration(Global.StreamThrottle) * time.Millisecond)
			}

		}

	}

	if fType == "text" {

		file, err := os.Open(fFilePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)

		if err != nil {
			fmt.Println(err)
			return
		}

		var text string

		if fPasswordHash != "" {

			err, iv, aesCTR, nonceSize := GetDecryptInfo(fFilePath, decryptFileHash)
			if err != nil {

				fmt.Println(err)

			}

			err, decrypted := DecryptFileStream(content[nonceSize:], len(content)-nonceSize, iv, aesCTR)
			if err != nil {
				fmt.Println(err)
				return
			}

			text = string(decrypted)

		} else {

			text = string(content)

		}

		//URL shortener
		_, err = url.ParseRequestURI(text)
		//if it's valid url
		if err == nil {

			http.Redirect(w, r, text, http.StatusSeeOther)

		} else {

			tmpl, err := template.ParseFS(templateFiles, "templates/pasteTemplate.html")
			if err != nil {
				fmt.Println(err)
				return
			}

			if raw == "1" {

				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(text))

			} else {

				err = tmpl.Execute(w, struct {
					Text string
					Burn string
				}{Text: text, Burn: fBurn})
				if err != nil {
					fmt.Println(err)
					return
				}

			}

		}

		if fBurn == "1" {

			file.Close()
			_, err = db.Exec("DELETE FROM data WHERE id = ?", path)
			if err != nil {
				fmt.Println(err)
				return
			}

			err = os.Remove(fFilePath)
			if err != nil {
				fmt.Println(err)
				return
			}

		}

	}

}
