package tracer

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

var (
	zipkinEndpoint            = flag.String("zipkinendpoint", os.Getenv("Zipkin_Endpoint"), "zipkin api end point")
	zipkinEnableDebug         = flag.Bool("zipkindebug", true, "zipkin enable debugging?")
	appHost, appPort, appName string
)

func SetGlobalTracer(host string, port string, name string) {
	flag.Parse()
	if *zipkinEndpoint == "" {
		*zipkinEndpoint = "http://192.168.99.100:32355"
	}
	appHost = host
	appPort = port
	appName = name

	opentracing.SetGlobalTracer(CreateTracer())
}

func GetGlobalTracer() opentracing.Tracer {
	return opentracing.GlobalTracer()
}

func CreateTracer() opentracing.Tracer {
	tracer, err := zipkin.NewTracer(
		CreateRecorder(CreateCollector()),
		zipkin.ClientServerSameSpan(true),
	)

	if err != nil {
		fmt.Errorf("error creating tracker %v", err)
	}

	return tracer
}

func CreateCollector() zipkin.Collector {
	url := *zipkinEndpoint + "/api/v1/spans"
	collector, err := zipkin.NewHTTPCollector(url)
	if err != nil {
		fmt.Errorf("error creating collector %v", err)
	}

	return collector
}

func CreateRecorder(collector zipkin.Collector) zipkin.SpanRecorder {
	log.Printf("url:%s | host: %s | port: %s | name: %s", *zipkinEndpoint, appHost, appPort, appName)
	return zipkin.NewRecorder(collector,
		*zipkinEnableDebug,
		appHost+appPort,
		appName)
}

func InjectSpan(httpReq *http.Request, ctx context.Context) *http.Request {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(httpReq.Header))
	}
	return httpReq
}

func CreateSpanFromRequest(req *http.Request, operationName string) opentracing.Span {
	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	tracer := GetGlobalTracer()
	extractedContext, err := tracer.Extract(
		opentracing.HTTPHeaders,
		carrier,
	)
	var span opentracing.Span

	if err != nil {
		span = tracer.StartSpan(operationName)
	} else {
		span = tracer.StartSpan(
			operationName,
			opentracing.ChildOf(extractedContext))
	}

	ext.SpanKindRPCServer.Set(span)
	span.SetTag("http.method", req.Method)
	span.SetTag("http.url", req.URL)

	return span
}

// Use it when the resource that you want to trace have no server side support
//wrap a call to an external service which is not instrumented
func StartRemoteServiceSpan(ctx opentracing.SpanContext, operationName string, remoteServiceName string, remoteServiceHost string, remoteServicePort uint16) opentracing.Span {
	span := opentracing.StartSpan(operationName, opentracing.ChildOf(ctx))
	ext.PeerHostname.Set(span, remoteServiceHost)
	ext.PeerPort.Set(span, remoteServicePort)
	ext.PeerService.Set(span, remoteServiceName)
	ext.SpanKind.Set(span, "resource")
	return span
}
