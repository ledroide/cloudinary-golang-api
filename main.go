package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ledroide/cloudinary-golang-api/handlers"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var addr = flag.String("listen-address", ":8090", "The address to listen on for HTTP requests.")

var zipkinEndpoint = flag.String("zipkinendpoint", os.Getenv("Zipkin_Endpoint"), "zipkin api end point")

const debugMode = true
const hostPort = "8090"

func main() {

	flag.Parse()

	routes := mux.NewRouter()
	routes.HandleFunc("/image", prometheus.InstrumentHandlerFunc("post_image", handlers.PostImageHandler)).Methods("POST")
	routes.HandleFunc("/image/{id}", prometheus.InstrumentHandlerFunc("get_image", handlers.GetImageHandler)).Methods("GET")
	routes.HandleFunc("/upload", prometheus.InstrumentHandlerFunc("upload_image", handlers.UploadImageHandler)).Methods("POST")
	routes.HandleFunc("/upload", prometheus.InstrumentHandlerFunc("get_upload_interface_image", handlers.GetUploadInterfaceHandler)).Methods("GET")
	routes.HandleFunc("/metrics", prometheus.Handler().ServeHTTP)
	prometheus.InstrumentHandler("routes", routes)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.Handle("/", routes)

	if *zipkinEndpoint == "" {
		flag.PrintDefaults()
		log.Println("The zipkin end point is missing. Please add the env. var 'Zipkin_Endpoint'")
		*zipkinEndpoint = "http://192.168.99.100:32355"
	}

	log.Println("The zipkin end point is: ", *zipkinEndpoint)
	zipkinHTTPEndpoint := *zipkinEndpoint + "/api/v1/spans"

	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)

	if err != nil {
		log.Printf("unable to create Zipkin HTTP collector: %+v", err)
	}

	recorder := zipkin.NewRecorder(collector, debugMode, hostPort, "imageService")

	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(true),
	)
	if err != nil {
		log.Printf("unable to create Zipkin tracer: %+v", err)
	}

	opentracing.SetGlobalTracer(tracer)

	srv := &http.Server{
		Handler:      TrackerHandler(tracer, routes),
		Addr:         *addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

}

func TrackerHandler(tracer opentracing.Tracer, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wireContext, err := tracer.Extract(
			opentracing.TextMap,
			opentracing.HTTPHeadersCarrier(r.Header),
		)
		if err != nil {
			fmt.Printf("error encountered while trying to extract span: %+v\n", err)
		}
		span := tracer.StartSpan(r.URL.Path, ext.RPCServerOption(wireContext))
		defer span.Finish()
		ctx := opentracing.ContextWithSpan(r.Context(), span)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}
