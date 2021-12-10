package chitracer

import (
	"net/http"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	ComponentName  string
	Tags           map[string]interface{}
	SpanFilter     func(r *http.Request) bool
}

func Tracer(tr opentracing.Tracer, trCfg *Config, opts ...nethttp.MWOption) func(next http.Handler) http.Handler {
	mwOpts := []nethttp.MWOption{
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "HTTP " + r.Method
		}),
	}

	if trCfg != nil {
		cfg := *trCfg
		mwOpts = append(mwOpts, nethttp.MWSpanObserver(func(span opentracing.Span, r *http.Request) {
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
			mwOpts = append(mwOpts, nethttp.MWComponentName(cfg.ComponentName))
		}
		if cfg.SpanFilter != nil {
			mwOpts = append(mwOpts, nethttp.MWSpanFilter(cfg.SpanFilter))
		}
	}
	mwOpts = append(mwOpts, opts...)

	return func(next http.Handler) http.Handler {
		return nethttp.Middleware(
			tr,
			next,
			mwOpts...,
		)
	}
}
