package main

import (
	"fmt"
	"gitlab.ciklum.net/ciklum-bpa/esb/app"
	"log"
	//	"time"
	"net/http"
//	"net/http/httputil"
	"strings"
//	"encoding/json"
)

//var ServicesProxy httputil.ReverseProxy

func EsbRequestsHandler(w http.ResponseWriter, r *http.Request, kernel *app.AppKernelStruct, esbSvc *app.EsbServiceClient, uriPattern *app.AppConfigRoutePattern) {
	baseUri := esbSvc.BaseUrl(r)
//	authRequest := kernel.Config
//	app.VarDump(authRequest)

//	w := *rw

	log.Printf("Handling %s %s by %s --> %s%s\n", r.Method, r.URL.Path, esbSvc.ServiceName, baseUri, uriPattern.Target)

	authenticated, err := kernel.Authenticate(r)
	if err != nil {
		log.Println("Authentication failed for", r.Method, r.URL.Path)
	}

	// allow cross domain AJAX requests
//	w.Header().Set("Content-Type", "application/json1; charset=utf-8")
//	w.Header().Set("Content-Type", "application/json2; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
//	w.Header().Set("Content-Type", "application/json3; charset=utf-8")
//
//	return w

	if authenticated != true {
		fmt.Println(r.Header.Get("Accept"));

		// the order of setting matters! Don't change it
		if strings.Contains(r.Header.Get("Accept"), "application/json") { // supports json
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{\"_error\":\"Unauthorized\"}"))
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		}

		return
	}

	w.Write([]byte("Hello!"))

}

func main() {
	kernel := app.Boot(true)

	for _, esbService := range *kernel.Services {
		//		app.VarDump("Service %v", esbService)
		//		app.VarDump("Service Config for service", esbService.ServiceConfig)
		//		app.VarDump("Service ROUTES", esbService.ServiceConfig.Routes)

		for _, uriPattern := range esbService.ServiceConfig.Routes {
			if uriPattern.Target == "" { // not set
				uriPattern.Target = uriPattern.Pattern
			}

			// making copies of values to ensure that referencing won't break things
			svc := *esbService
			pattern := uriPattern

			http.HandleFunc(uriPattern.Pattern, func(w http.ResponseWriter, r *http.Request) {
//				w.Header().Set("Content-Type", "application/pdf; charset=utf-8")
//				w.Header().Set("Content-Type", "application/x; charset=utf-8")
				EsbRequestsHandler(w, r, kernel, &svc, &pattern)
			})

		}
	}

//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
//		w.Header().Set("x-CUSTOM", "header")
//		fmt.Fprintf(w, "I went to %q", r.URL.Path)
//	})

	dsl := fmt.Sprintf(":%d", 80)
	log.Println("Started listening at", dsl)
	log.Fatal(http.ListenAndServe(dsl, nil))
}
