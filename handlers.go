package stream

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"syscall"

	"github.com/dustin/go-humanize"
	"github.com/julienschmidt/httprouter"
)

// Handler is the main handler for Stream.
func (h Stream) Handler() http.Handler {
	r := httprouter.New()

	r.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("DAV", "1")
		w.Header().Set("MS-Author-Via", "DAV")
	})

	r.Handle("PROPFIND", "/", h.propRoot)
	r.Handle("PROPFIND", "/films", h.propFilms)
	r.Handle("PROPFIND", "/shows", h.propShows)

	r.Handle("PROPFIND", "/films/:file", h.addFile(h.propFile))
	r.Handle("GET", "/films/:file", addRequestID(h.addFile(h.streamFile)))
	r.Handle("HEAD", "/films/:file", addRequestID(h.addFile(h.streamFile)))

	r.Handle("PROPFIND", "/shows/:folder", h.propEpisodes)
	r.Handle("PROPFIND", "/shows/:folder/:file", h.addFile(h.propFile))
	r.Handle("GET", "/shows/:folder/:file", addRequestID(h.addFile(h.streamFile)))
	r.Handle("HEAD", "/shows/:folder/:file", addRequestID(h.addFile(h.streamFile)))

	return r
}

func writeXML(w http.ResponseWriter, responses []Response) {
	res := MultiStatus{
		Namespace: "DAV:",
		Responses: responses,
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)

	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(res)
}

// propRoot creates a PROPFIND response with a `films` and `shows` folder.
//
// Does not require any middleware.
func (h *Stream) propRoot(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	responses := []Response{
		createDavFolder("/", ""),
		createDavFolder("/films/", "films"),
		createDavFolder("/shows/", "shows"),
	}

	writeXML(w, responses)
}

// propFilms creates a PROPFIND response with all the films in the datastore.
//
// Does not require any middleware.
func (h Stream) propFilms(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	responses := []Response{createDavFolder("/films/", "films")}

	films, err := h.store.RecursiveFiles(r.Context(), h.filmsID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	for _, f := range films {
		name := fileWithID(f.Name, f.ID)
		responses = append(responses, createDavFile("/films/"+url.PathEscape(name), f))
	}

	writeXML(w, responses)
}

// propShows creates a PROPFIND response with all the shows in the datastore.
//
// Does not require any middleware.
func (h Stream) propShows(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	responses := []Response{createDavFolder("/shows/", "shows")}

	shows, err := h.store.RecursiveFolders(r.Context(), h.showsID, h.depth)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	for _, f := range shows {
		folderPath := "/shows/" + url.PathEscape(folderWithID(f.Name, f.ID))
		responses = append(responses, createDavFolder(folderPath, f.Name))
	}

	writeXML(w, responses)
}

// propEpisodes creates a PROPFIND response with all episodes of the show.
//
// Does not require any middleware.
func (h Stream) propEpisodes(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	folder, id, err := folderIDFromName(ps.ByName("folder"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	responses := []Response{createDavFolder(r.URL.String(), folder)}

	episodes, err := h.store.RecursiveFiles(r.Context(), id)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}

	for _, f := range episodes {
		fileName := fileWithID(f.Name, f.ID)
		filePath := r.URL.String() + "/" + url.PathEscape(fileName)
		responses = append(responses, createDavFile(filePath, f))
	}

	writeXML(w, responses)
}

// propFile creates a PROPFIND response for the given File in context.
//
// Requires the `addFile` middleware.
func (h Stream) propFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	f := getFile(r.Context())

	responses := []Response{
		createDavFile(r.URL.String(), f),
	}

	writeXML(w, responses)
}

// streamFile fetches chunks of the file in Google Drive until the request is closed or hits EOF.
//
// Requires the `addFile` middleware.
func (h Stream) streamFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := getRequestID(r.Context())
	f := getFile(r.Context())

	startPos, endPos, err := parseRange(r.Header.Get("Range"), uint64(f.Size))
	if err != nil {
		startPos = 0
		endPos = uint64(f.Size) - 1
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startPos, endPos, f.Size))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", endPos-startPos+1))
	w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(f.Name)))
	w.WriteHeader(http.StatusPartialContent)

	if r.Method != "GET" {
		return
	}

	fmt.Printf("%s - request: %s\n", requestID, r.Header.Get("Range"))

	go func() {
		<-r.Context().Done()

		fmt.Printf("%s - stream closed\n", requestID)
	}()

	const metaBuffer = 10 * 1024 * 1024
	const buffer = 50 * 1024 * 1024
	chunkStart := startPos

	for {
		if chunkStart >= endPos {
			break
		}

		chunkEnd := chunkStart + buffer
		if chunkStart == 0 {
			chunkEnd = chunkStart + metaBuffer
		}

		if chunkEnd > endPos {
			chunkEnd = endPos
		}

		fmt.Printf("%s - chunk: %d -> %d (%s)\n", requestID, chunkStart, chunkEnd, humanize.Bytes(chunkEnd-chunkStart))

		err = h.fetch.Range(r.Context(), w, f.ID, chunkStart, chunkEnd)
		if err == nil {
			chunkStart = chunkEnd + 1
			continue
		}

		if errors.Is(err, syscall.EPIPE) {
			fmt.Printf("%s - stream epipe\n", requestID)
			break
		}

		if errors.Is(err, context.Canceled) {
			fmt.Printf("%s - context cancelled\n", requestID)
			break
		}

		fmt.Printf("%s - error: %v\n", requestID, err)
		break
	}
}
