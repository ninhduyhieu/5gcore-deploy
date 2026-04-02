package httpdp

import (
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/httpw"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// a wrapper for gin.Context. It implements the sbi.RequestContext
// abstraction
type ginContextEx struct {
	context *gin.Context
	written bool
}

func newGinContextEx(ctx *gin.Context) *ginContextEx {
	ret := &ginContextEx{
		context: ctx,
	}
	return ret
}

func (c *ginContextEx) Param(key string) string {
	return c.context.Param(key)
}

func (c *ginContextEx) Header(key string) string {
	return c.context.Request.Header.Get(key)
}

func (c *ginContextEx) RequestBody() (int64, io.Reader) {
	return c.context.Request.ContentLength, c.context.Request.Body
}

//TODO: add headers argument
func (c *ginContextEx) WriteResponse(code int, ie sbi.SbiIE, headers map[string]string) {
	c.written = true
	if ie == nil {
		c.context.JSON(code, nil)
		return
	}
	if contentLength, content, err := sbi.Encode(ie); err != nil {
		c.context.JSON(http.StatusInternalServerError, &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Encode response body failed: %+v", err),
		})
	} else {
		c.context.DataFromReader(code, contentLength, "application/json", content, headers)
	}
}

//build sbi route into http routes with http request handler
func MakeHttpRoutes[T any](routes []sbi.Route[T], appHandler T) []httpw.Route {
	ret := make([]httpw.Route, len(routes), len(routes))
	for i, r := range routes {
		ret[i] = httpw.Route{
			Method:  r.Method,
			Pattern: r.Path,
			Handler: createGinHandler(r.Handler, appHandler),
		}
	}
	return ret
}

// build a service handler for gin
// the first parameter is the sbi handler, the second parameter is an
// NF specific producer implementation.

func createGinHandler[T any](sbiHandler func(sbi.RequestContext, T), appHandler T) gin.HandlerFunc {
	return func(context *gin.Context) {
		ctx := newGinContextEx(context)
		//call the sbi producer handler to decode request and call
		//application handler to process the request
		sbiHandler(ctx, appHandler)
		if !ctx.written {
			ctx.context.JSON(http.StatusInternalServerError, &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Unspecified error",
			})
		}
	}
}
