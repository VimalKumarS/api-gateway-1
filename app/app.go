package app

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"time"
)

const SvcServiceTimeout = 360 // timeout for service connection
const SvcKeepAlive = 60000    // service client keep-alive settings

// used for AppConfigStruct - simple type key-values
type KeyValueType map[string]string
type AppConfigRoutePattern struct {
	Pattern string
	Target  string "target,omitempty" // if empty, use pattern as it is
}

// allows mixed types of simple type key-values and sub structures
type AppConfigRoutingRouteStruct struct {
	Timeout       int            "timeout,omitempty"
	KeepAlive     int            "keep_alive,omitempty"
	CustomHeaders []KeyValueType "custom_headers,omitempty"
	Instances     []string
	Routes        []AppConfigRoutePattern "routes"
}

type AppConfigRoutingStruct map[string]AppConfigRoutingRouteStruct

type AppConfigEsbStruct struct {
	Scheme string
	Host   string
	Port   int
}

type AppConfigLoggingStruct struct {
	File   string
	Rotate string
	Level  string
}

type AppConfigTrustedSourcesType []string

type AppConfigSecurityStruct struct {
	PublicEndpoints     []string                      "public_endpoints,omitempty"
	UserTokenValidation string                        "user_token_validation"
	TrustedSources      []AppConfigTrustedSourcesType "trusted_sources,omitempty"
	ServiceAliasName    string "service_alias"
}

type AppConfigStruct struct {
	Debug       bool
	Environment string "app_env"
	Esb         AppConfigEsbStruct
	Logging     AppConfigLoggingStruct
	Security    AppConfigSecurityStruct
	Routing     AppConfigRoutingStruct
}

type EsbServiceClient struct {
	Client        *http.Client
	ServiceName   string
	ServiceConfig AppConfigRoutingRouteStruct
	//	ServiceConfig   *AppConfigRoutingRouteStruct // WHY IT DOES NOT WORK?
	currentInstance int
}

// BaseUrl get instance base URL based on request data and load balancer
func (svc *EsbServiceClient) BaseUrl(r *http.Request) string {
	arrSize := len(svc.ServiceConfig.Instances)
	if arrSize > 1 {
		if arrSize <= svc.currentInstance+1 {
			svc.currentInstance = 0
		} else {
			svc.currentInstance++
		}
		// TODO: implement randomizer/round-robin aka load balancer to select instance with check of health status
	}

	return svc.ServiceConfig.Instances[svc.currentInstance]
}

// TargetUrl get targetted URL without base, based on given requested URL
// Returns nil if nothing was found
func (svc *EsbServiceClient) TargetUrl(u *url.URL) (string, error) {
	for _, uriPattern := range svc.ServiceConfig.Routes {

		if uriPattern.Pattern == u.Path {
			if uriPattern.Target == "" { // not set
				return uriPattern.Pattern, nil
			}

			return uriPattern.Target, nil
		}
	}

	return "", errors.New(fmt.Sprint("Not found path:", u.Path))
}


// Authenticate checks permissions to execute given request
func (kernel *AppKernelStruct) Authenticate(r *http.Request) (bool, error) {

	if kernel.IsPublicRequest(r) {
		return true, nil
	}
//	authRequest := http.Req

	log.Println("Error occurred during authentication: ", r)
	return false, errors.New("Error occurred during authentication")
}

// IsPublicRequest returns true if given request leads to public resources and don't need authorization
func (kernel *AppKernelStruct) IsPublicRequest(r *http.Request) (bool) {
	for _, path := range kernel.Config.Security.PublicEndpoints {
		if path == r.URL.Path {
			return true
		}
	}

	return false
}

type EsbServiceClientCollection map[string]*EsbServiceClient

// config for the running application
var AppConfig AppConfigStruct

type AppKernelStruct struct {
	Config   *AppConfigStruct
	Services *EsbServiceClientCollection
}

// Init application
func Boot(debug bool) *AppKernelStruct {
	// set up debug mode
	AppConfig.Debug = debug
	if debug {
		fmt.Println("Running in application DEBUG mode")
	}

	// convert relative path to absolute for file use
	configAbs, err := filepath.Abs("app/config/parameters.yml")
	AssertNil(err, "Failed to define absolute path of config")

	// read contents of file to temp string
	configContents, err := ioutil.ReadFile(configAbs)
	AssertNil(err, "Failed to read config")

	// parse contents to YAML format and put results to AppConfig
	err = yaml.Unmarshal(configContents, &AppConfig)
	AssertNil(err, "Failed to parse config")

	// some debug info..
	if debug {
		fmt.Printf("Environment: %v\n", AppConfig.Environment)
		fmt.Printf("Server: %s://%s:%d/\n", AppConfig.Esb.Scheme, AppConfig.Esb.Host, AppConfig.Esb.Port)
	}

	kernel := &AppKernelStruct{Config: &AppConfig}
	kernel.InitHttpClients()

	return kernel
}

// init http clients for further usage
func (kernel *AppKernelStruct) InitHttpClients() *EsbServiceClientCollection {
	config := kernel.Config
	AssertTrue(len(config.Routing) > 0, "Did not find any routes for the server")

	esbServiceCollection := make(EsbServiceClientCollection, len(config.Routing))

	for svcName, svcCfg := range config.Routing {
		VarDump(fmt.Sprintf("Adding HTTP client for \"%s\"", svcName))

		// set defaults if necessary
		if svcCfg.Timeout <= 0 {
			svcCfg.Timeout = SvcServiceTimeout
		}

		if svcCfg.KeepAlive <= 0 {
			svcCfg.KeepAlive = SvcKeepAlive
		}

		// init ESB service instance
		esbService := new(EsbServiceClient)
		// TODO: configure Transport and other params for increase of performance. See RoundTripper...
		// TODO: TLSClientConfig:    &tls.Config{RootCAs: pool},
		// TODO: DisableCompression: true,
		esbService.Client = &http.Client{
			Timeout: time.Duration(svcCfg.Timeout) * time.Second,
		}
		esbService.ServiceName = svcName
		esbService.ServiceConfig = svcCfg
		//		esbService.ServiceConfig = &config.Routing[svcName]
		//		esbService.ServiceConfig = &svcCfg

		esbServiceCollection[svcName] = esbService
	}

	kernel.Services = &esbServiceCollection

	return kernel.Services
}

// Throws panic fatal exception if value is different from nil. Used for "err" var status check
func AssertNil(v interface{}, msg string) {
	if v != nil {
		log.Print(v)
		log.Fatal(msg)
	}
}

// Throws panic fatal exception if value is different from true
func AssertTrue(v bool, msg string) {
	if !v {
		log.Fatal(msg)
	}
}

// dump debug values if in DEBUG mode
func VarDump(values ...interface{}) {
	if AppConfig.Debug { // TODO: instead of streaming, use logging method?
		for _, value := range values {
			fmt.Println(value)
		}
	}
}

type JsonError struct {
	Error string
}
