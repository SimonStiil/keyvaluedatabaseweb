package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type Application struct {
	Config       ConfigType
	KVDBClient   *Client
	Logger       *slog.Logger
	Requestcount int
}
type KeyValueList struct {
	Api       string
	Namespace string
	Items     []KeyValue
}
type KeyValue struct {
	Id       int
	Key      string
	Value    string
	Lines    int
	ReadOnly bool
}

type NamespaceKeyValueList struct {
	Api   string
	Items []NamespaceKeyValue
}
type NamespaceKeyValue struct {
	Id     int
	Name   string
	Size   int
	Access bool
}

func (App *Application) HealthActuator(w http.ResponseWriter, r *http.Request) {
	logger := App.Logger.With(slog.Any("id", RandomID())).With(slog.Any("function", "HealthActuator")).With(slog.Any("struct", "Application")).With(slog.Any("remoteAddr", r.RemoteAddr)).With(slog.Any("method", r.Method))
	if App.Config.Prometheus.Enabled {
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method, "").Inc()
	}
	if !(r.URL.Path == "/system/health") {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	var reply Health
	if App.KVDBClient.GetHealth(logger) != nil {
		reply.Status = "UP"
		logger.Info("health", "status", http.StatusOK)
	} else {
		reply.Status = "DOWN"
		w.WriteHeader(http.StatusInternalServerError)
		logger.Info("health", "status", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
}

func (App *Application) setupLogging() {
	logLevel := strings.ToLower(App.Config.Logging.Level)
	logFormat := strings.ToLower(App.Config.Logging.Format)
	loggingLevel := new(slog.LevelVar)
	switch logLevel {
	case "debug":
		loggingLevel.Set(slog.LevelDebug)
	case "warn":
		loggingLevel.Set(slog.LevelWarn)
	case "error":
		loggingLevel.Set(slog.LevelError)
	default:
		loggingLevel.Set(slog.LevelInfo)
	}
	switch logFormat {
	case "json":
		App.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel}))
	default:
		App.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel}))
	}
	App.Logger.Info("Logging started with options", "format", App.Config.Logging.Format, "level", App.Config.Logging.Level, "function", "setupLogging")
	slog.SetDefault(App.Logger)
}

func (App *Application) RootController(w http.ResponseWriter, r *http.Request) {
	request := GetRequestParameters(r)
	logger := App.Logger.With(slog.Any("id", request.ID)).With(slog.Any("remoteAddr", r.RemoteAddr)).With(slog.Any("method", r.Method), "path", r.URL.EscapedPath())
	logger.Debug("Root Request", "function", "HealthActuator", "struct", "Application")
	if request.Api == "" {
		http.Redirect(w, r, "/v1", http.StatusSeeOther)
		return
	}
	if request.Api == "v1" {
		if request.Namespace != "" {
			App.KeysController(w, request)
			return
		} else {
			App.NamespaceController(w, request)
			return
		}
	}
	logger.Info("PathNotFound", "status", http.StatusNotFound)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(fmt.Sprintf("%v Not Found", http.StatusNotFound)))
}

func (App *Application) NamespaceController(w http.ResponseWriter, request *RequestParameters) {
	logger := App.Logger.With(slog.Any("id", request.ID)).With(slog.Any("remoteAddr", request.orgRequest.RemoteAddr)).With(slog.Any("method", request.Method), "path", request.Path)
	debugLogger := logger.With(slog.Any("function", "NamespaceController")).With(slog.Any("struct", "Application"))
	debugLogger.Debug("Namespace Request")
	statuscode := http.StatusOK
	requests.WithLabelValues(request.Path, request.Method, "").Inc()
	logger.Info("Keys request", "status", statuscode)
	kvlist, err := App.KVDBClient.GetNamespaceList(logger)
	if err != nil {
		debugLogger.Debug("GetNamespaceList Error", "type", fmt.Sprintf("%t", err), "error", err)
		App.BadRequestHandler(logger, w, request)
		return
	}
	KeyValueList := App.convertNamespaceList(request.Api, kvlist)
	w.WriteHeader(statuscode)
	// https://pkg.go.dev/html/template
	tmpl := template.Must(template.ParseFiles("namespaceindex.html"))
	tmpl.Execute(w, KeyValueList)
}

func (App *Application) KeysController(w http.ResponseWriter, request *RequestParameters) {
	logger := App.Logger.With(slog.Any("id", request.ID)).With(slog.Any("remoteAddr", request.orgRequest.RemoteAddr)).With(slog.Any("method", request.Method), "path", request.Path)
	debugLogger := logger.With(slog.Any("function", "KeysController")).With(slog.Any("struct", "Application"))
	debugLogger.Debug("Keys Request")
	statuscode := http.StatusOK
	if request.Method == "POST" {
		err := request.orgRequest.ParseForm()
		if err != nil {
			debugLogger.Debug("ParseForm Error", "type", fmt.Sprintf("%t", err), "error", err)
			App.BadRequestHandler(logger, w, request)
			return
		} else {
			debugLogger.Debug("ParseForm", "values", request.orgRequest.PostForm)
		}
		function := request.orgRequest.PostFormValue("input")
		requests.WithLabelValues(request.Path, request.Method, function).Inc()
		pair := rest.KVPairV2{Key: request.orgRequest.PostFormValue("key"),
			Value: request.orgRequest.PostFormValue("value"), Namespace: request.Namespace}
		switch function {
		case "Create", "Update":
			err = App.KVDBClient.SetKey(logger, request.Namespace, pair)
		case "Generate":
			err = App.KVDBClient.Generate(logger, request.Namespace, pair.Key)
		case "Roll":
			err = App.KVDBClient.Roll(logger, request.Namespace, pair.Key)
		case "Delete":
			err = App.KVDBClient.DeleteKey(logger, request.Namespace, pair.Key)
		default:
			log.Printf("I RootController Unknown value %v", function)

		}
		if err != nil {
			debugLogger.Debug("Post Function Error", "type", fmt.Sprintf("%t", err), "error", err)
			App.BadRequestHandler(logger, w, request)
			return
		}
		logger.Info("Keys request", "status", statuscode)
	} else {
		requests.WithLabelValues(request.Path, request.Method, "").Inc()
		logger.Info("Keys request", "status", statuscode)
	}
	kvlist, err := App.KVDBClient.GetKeyList(logger, request.Namespace)
	if err != nil {
		debugLogger.Debug("GetKeyList Error", "type", fmt.Sprintf("%t", err), "error", err)
		App.BadRequestHandler(logger, w, request)
		return
	}
	KeyValueList := App.convertKeyList(request.Api, request.Namespace, kvlist)
	w.WriteHeader(statuscode)
	// https://pkg.go.dev/html/template
	tmpl := template.Must(template.ParseFiles("keysindex.html"))
	tmpl.Execute(w, KeyValueList)
}

func (App *Application) countRune(s string, r rune) int {
	count := 1
	for _, c := range s {
		if c == r {
			count++
		}
	}
	return count
}

func (App *Application) convertKeyList(api string, namespace string, list []rest.KVPairV2) KeyValueList {
	systemNS := namespace == "kvdb"
	kvList := KeyValueList{Api: api, Namespace: namespace}
	for i, pair := range list {
		readOnly := systemNS && pair.Key == "counter"
		kvList.Items = append(kvList.Items, KeyValue{Id: i, Key: pair.Key, Value: pair.Value, Lines: App.countRune(pair.Value, '\n'), ReadOnly: readOnly})
	}
	return kvList
}

func (App *Application) convertNamespaceList(api string, list []rest.NamespaceV2) NamespaceKeyValueList {
	namespaceKeyValueList := NamespaceKeyValueList{Api: api}
	for i, pair := range list {
		namespaceKeyValueList.Items = append(namespaceKeyValueList.Items, NamespaceKeyValue{Id: i, Name: pair.Name, Size: pair.Size, Access: pair.Access})
	}
	return namespaceKeyValueList
}

func (App *Application) BadRequestHandler(logger *slog.Logger, w http.ResponseWriter, request *RequestParameters) {
	logger.Info("Bad Request", "status", 400)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 Bad Request"))
}
