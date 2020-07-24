package stream

import (
	"context"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	ds "github.com/m-rots/bernard/datastore"
	"github.com/rs/xid"
)

type ctxKey int

const (
	requestIDKey = ctxKey(0)
	fileKey      = ctxKey(1)
)

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func getRequestID(ctx context.Context) string {
	return ctx.Value(requestIDKey).(string)
}

func addRequestID(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		guid := xid.New()
		ctx := withRequestID(r.Context(), guid.String())
		next(w, r.WithContext(ctx), ps)
	}
}

func withFile(ctx context.Context, file ds.File) context.Context {
	return context.WithValue(ctx, fileKey, file)
}

func getFile(ctx context.Context) ds.File {
	return ctx.Value(fileKey).(ds.File)
}

func (h Stream) addFile(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		name := ps.ByName("file")
		id, err := fileIDFromName(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		f, err := h.store.GetFile(r.Context(), id)
		if err != nil {
			fmt.Println(err)
			http.NotFound(w, r)
			return
		}

		ctx := withFile(r.Context(), f)
		next(w, r.WithContext(ctx), ps)
	}
}
