package gintracer

// Apache 2.0 - https://github.com/opentracing-contrib/go-stdlib
// REF - https://github.com/opentracing-contrib/go-stdlib/blob/master/nethttp/server.go

// Also
// BSD 3-Clause "New" or "Revised" License
//
// REF - https://github.com/opentracing-contrib/go-gin

/*
Copyright (c) 2018, opentracing-contrib
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of go-gin nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const defaultComponentName = "net/http"
const responseSizeKey = "http.response_size"

type mwOptions struct {
	opNameFunc    func(r *http.Request) string
	spanFilter    func(r *http.Request) bool
	spanObserver  func(span opentracing.Span, r *http.Request)
	urlTagFunc    func(u *url.URL) string
	componentName string
}

// MWOption controls the behavior of the Middleware.
type MWOption func(*mwOptions)

// OperationNameFunc returns a MWOption that uses given function f
// to generate operation name for each server-side span.
func OperationNameFunc(f func(r *http.Request) string) MWOption {
	return func(options *mwOptions) {
		options.opNameFunc = f
	}
}

// MWComponentName returns a MWOption that sets the component name
// for the server-side span.
func MWComponentName(componentName string) MWOption {
	return func(options *mwOptions) {
		options.componentName = componentName
	}
}

// MWSpanFilter returns a MWOption that filters requests from creating a span
// for the server-side span.
// Span won't be created if it returns false.
func MWSpanFilter(f func(r *http.Request) bool) MWOption {
	return func(options *mwOptions) {
		options.spanFilter = f
	}
}

// MWSpanObserver returns a MWOption that observe the span
// for the server-side span.
func MWSpanObserver(f func(span opentracing.Span, r *http.Request)) MWOption {
	return func(options *mwOptions) {
		options.spanObserver = f
	}
}

// MWURLTagFunc returns a MWOption that uses given function f
// to set the span's http.url tag. Can be used to change the default
// http.url tag, eg to redact sensitive information.
func MWURLTagFunc(f func(u *url.URL) string) MWOption {
	return func(options *mwOptions) {
		options.urlTagFunc = f
	}
}

// Middleware is a gin native version of the equivalent middleware in:
//   https://github.com/opentracing-contrib/go-stdlib/
func Middleware(tr opentracing.Tracer, options ...MWOption) gin.HandlerFunc {
	opts := mwOptions{
		opNameFunc: func(r *http.Request) string {
			return "HTTP " + r.Method
		},
		spanFilter:   func(r *http.Request) bool { return true },
		spanObserver: func(span opentracing.Span, r *http.Request) {},
		urlTagFunc: func(u *url.URL) string {
			return u.String()
		},
	}
	for _, opt := range options {
		opt(&opts)
	}

	return func(c *gin.Context) {
		if !opts.spanFilter(c.Request) {
			c.Next()
			return
		}

		carrier := opentracing.HTTPHeadersCarrier(c.Request.Header)
		ctx, _ := tr.Extract(opentracing.HTTPHeaders, carrier)
		op := opts.opNameFunc(c.Request)
		sp := tr.StartSpan(op, ext.RPCServerOption(ctx))
		ext.HTTPMethod.Set(sp, c.Request.Method)
		ext.HTTPUrl.Set(sp, opts.urlTagFunc(c.Request.URL))
		opts.spanObserver(sp, c.Request)

		// set component name, use "net/http" if caller does not specify
		componentName := opts.componentName
		if componentName == "" {
			componentName = defaultComponentName
		}
		ext.Component.Set(sp, componentName)
		c.Request = c.Request.WithContext(
			opentracing.ContextWithSpan(c.Request.Context(), sp))

		mt := &metricsTracker{ResponseWriter: c.Writer}

		defer func() {
			panicErr := recover()
			didPanic := panicErr != nil

			if mt.status == 0 && !didPanic {
				// Standard behavior of http.Server is to assume status code 200 if one was not written by a handler that returned successfully.
				// https://github.com/golang/go/blob/fca286bed3ed0e12336532cc711875ae5b3cb02a/src/net/http/server.go#L120
				mt.status = 200
			}
			if mt.status > 0 {
				ext.HTTPStatusCode.Set(sp, uint16(mt.status))
			}
			if mt.size > 0 {
				sp.SetTag(responseSizeKey, mt.size)
			}
			if mt.status >= http.StatusInternalServerError || didPanic {
				ext.Error.Set(sp, true)
			}
			sp.Finish()

			if didPanic {
				panic(panicErr)
			}
		}()

		c.Next()
	}
}

type Config struct {
	ServiceName    string
	ServiceVersion string
	ComponentName  string
	Tags           map[string]interface{}
	SpanFilter     func(r *http.Request) bool
}

func Mid(tr opentracing.Tracer, trCfg *Config, opts ...MWOption) gin.HandlerFunc {
	mwOpts := []MWOption{}
	if trCfg != nil {
		cfg := *trCfg
		mwOpts = append(mwOpts, MWSpanObserver(func(span opentracing.Span, r *http.Request) {
			if cfg.ServiceName != "" {
				span.SetTag("service.name", cfg.ServiceName)
			}
			if cfg.ServiceVersion != "" {
				span.SetTag("version", cfg.ServiceVersion)
			}

			for k, v := range cfg.Tags {
				span.SetTag(k, v)
			}
		}))
		if cfg.ComponentName != "" {
			mwOpts = append(mwOpts, MWComponentName(cfg.ComponentName))
		}
		if cfg.SpanFilter != nil {
			mwOpts = append(mwOpts, MWSpanFilter(cfg.SpanFilter))
		}
	}
	mwOpts = append(mwOpts, opts...)
	return Middleware(tr, mwOpts...)
}
