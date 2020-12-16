package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xe/rhea/gemini"
)

type FileServer struct {
	Root      string `json:"root"`
	UserPaths bool   `json:"user_paths"`
	AutoIndex bool   `json:"auto_index"`
}

func (f FileServer) writeIndex(path string, r *gemini.Request, w gemini.ResponseWriter) {
	dir, err := os.Open(path)
	if err != nil {
		w.Status(gemini.StatusNotFound, err.Error())
		return
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		w.Status(gemini.StatusPermanentFailure, err.Error())
		return
	}

	sort.Strings(names)

	w.Status(gemini.StatusSuccess, "text/gemini")

	fmt.Fprintf(w, "# %s\n", r.URL.Path)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "=> .. ..")

	for _, name := range names {
		fmt.Fprintf(w, "=> ./%[1]s %[1]s", name)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Served by rhea")
}

func (f FileServer) serveFile(path string, w gemini.ResponseWriter) {
	fin, err := os.Open(path)
	if err != nil {
		w.Status(gemini.StatusTemporaryFailure, "can't open file")
		log.Printf("%v", err)
		return
	}
	defer fin.Close()

	mimeT := mime.TypeByExtension(filepath.Ext(path))
	w.Status(gemini.StatusSuccess, mimeT)
	io.Copy(w, fin)
}

func (f FileServer) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	path := filepath.Join(f.Root, r.URL.Path)
	st, err := os.Stat(path)
	if err != nil {
		w.Status(gemini.StatusNotFound, fmt.Sprint("can't find ", r.URL.Path))
		log.Printf("%v", err)
		return
	}

	if st.IsDir() {
		if !strings.HasSuffix(r.URL.Path, "/") {
			w.Status(gemini.StatusRedirectPermanent, fmt.Sprintf("%s/", r.URL.Path))
			return
		}
		newPath := filepath.Join(path, "index.gmi")
		_, err := os.Stat(newPath)
		if err != nil {
			if f.AutoIndex {
				f.writeIndex(path, r, w)
				return
			}
			w.Status(gemini.StatusNotFound, "this is a folder, but has no index")
			return
		}
		path = newPath
	}

	f.serveFile(path, w)
}
