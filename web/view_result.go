package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/blend/go-sdk/env"
	"github.com/blend/go-sdk/ex"
)

// ViewResult is a result that renders a view.
type ViewResult struct {
	ViewName   string
	StatusCode int
	ViewModel  interface{}
	Views      *ViewCache
	Template   *template.Template
}

// Render renders the result to the given response writer.
func (vr *ViewResult) Render(ctx *Ctx) (err error) {
	// you must set the template to be rendered.
	if vr.Template == nil {
		err = ex.New(ErrUnsetViewTemplate)
		return
	}

	if ctx.Tracer != nil {
		if typed, ok := ctx.Tracer.(ViewTracer); ok {
			tf := typed.StartView(ctx, vr)
			defer func() {
				tf.FinishView(ctx, vr, err)
			}()
		}
	}

	ctx.Response.Header().Set(HeaderContentType, ContentTypeHTML)

	// use a pooled buffer if possible
	var buffer *bytes.Buffer
	if vr.Views != nil && vr.Views.BufferPool != nil {
		buffer = vr.Views.BufferPool.Get()
		defer vr.Views.BufferPool.Put(buffer)
	} else {
		buffer = bytes.NewBuffer(nil)
	}

	err = vr.Template.Execute(buffer, &ViewModel{
		Env:       env.Env(),
		Ctx:       ctx,
		ViewModel: vr.ViewModel,
	})
	if err != nil {
		err = ex.New(err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte(fmt.Sprintf("%+v\n", err)))
		return
	}

	ctx.Response.WriteHeader(vr.StatusCode)
	_, err = ctx.Response.Write(buffer.Bytes())
	if err != nil {
		err = ex.New(err)
	}
	return
}
