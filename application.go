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

type KeyValue struct {
	Id    int
	Key   string
	Value string
	Lines int
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
	if App.KVDBClient.GetHealth() != nil {
		reply.Status = "UP"
		logger.Info("health", "status", http.StatusOK)
	} else {
		reply.Status = "DOWN"
		w.WriteHeader(http.StatusInternalServerError)
		logger.Info("health", "status", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
	return
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
	}
	if request.Api == "v1" {
		if request.Namespace != "" {
			App.KeysController(w, request)
		} else {
			App.NamespaceController(w, request)
		}
	}
	logger.Info("PathNotFound", http.StatusNotFound)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(fmt.Sprintf("%v Not Found", http.StatusNotFound)))
}

func (App *Application) NamespaceController(w http.ResponseWriter, request *RequestParameters) {
	statuscode := http.StatusOK
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			if App.Config.Debug {
				log.Printf("ParseForm: %v, %t\n", err, err)
			}
			App.BadRequestHandler().ServeHTTP(w, r)
		}
		function := r.PostFormValue("input")
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method, function).Inc()
		pair := rest.NamespaceV2{Name: r.PostFormValue("key")}
		var ok bool
		switch function {
		case "Create", "Update":
			ok = App.KVDBClient.Set(pair)
		case "Generate":
			ok = App.KVDBClient.Generate(pair.Key)
		case "Roll":
			ok = App.KVDBClient.Roll(pair.Key)
		case "Delete":
			ok = App.KVDBClient.DeleteNamespace(namespace)
		default:
			log.Printf("I RootController Unknown value %v", function)

		}
		if !ok {
			statuscode = http.StatusBadRequest
		}
		log.Printf("I %v %v %v %+v", r.Method, r.URL.Path, statuscode, r.PostForm)
	} else {
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method, "").Inc()
		log.Printf("I %v %v %v", r.Method, r.URL.Path, statuscode)
	}
	kvlist := App.KVDBClient.GetNamespaceList()
	KeyValueList := App.convertNamespaceList(kvlist)
	w.WriteHeader(statuscode)
	// https://pkg.go.dev/html/template
	tmpl := template.Must(template.ParseFiles("namespaceindex.html"))
	tmpl.Execute(w, KeyValueList)
}

func (App *Application) KeysController(w http.ResponseWriter, request *RequestParameters) {
	statuscode := http.StatusOK
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			if App.Config.Debug {
				log.Printf("ParseForm: %v, %t\n", err, err)
			}
			App.BadRequestHandler().ServeHTTP(w, r)
		}
		function := r.PostFormValue("input")
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method, function).Inc()
		pair := rest.KVPairV2{Key: r.PostFormValue("key"),
			Value: r.PostFormValue("value")}
		var ok bool
		switch function {
		case "Create", "Update":
			ok = App.KVDBClient.SetKey(namespace, pair)
		case "Generate":
			ok = App.KVDBClient.Generate(namespace, pair.Key)
		case "Roll":
			ok = App.KVDBClient.Roll(namespace, pair.Key)
		case "Delete":
			ok = App.KVDBClient.DeleteKey(namespace, pair.Key)
		default:
			log.Printf("I RootController Unknown value %v", function)

		}
		if !ok {
			statuscode = http.StatusBadRequest
		}
		log.Printf("I %v %v %v %+v", r.Method, r.URL.Path, statuscode, r.PostForm)
	} else {
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method, "").Inc()
		log.Printf("I %v %v %v", r.Method, r.URL.Path, statuscode)
	}
	kvlist := App.KVDBClient.GetList("")
	KeyValueList := App.convertKeyList(kvlist)
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

func (App *Application) convertKeyList(list []rest.KVPairV2) []KeyValue {
	var KeyValueList []KeyValue
	for i, pair := range list {
		KeyValueList = append(KeyValueList, KeyValue{Id: i, Key: pair.Key, Value: pair.Value, Lines: App.countRune(pair.Value, '\n')})
	}
	return KeyValueList
}

func (App *Application) convertNamespaceList(list []rest.NamespaceV2) []NamespaceKeyValue {
	var KeyValueList []NamespaceKeyValue
	for i, pair := range list {
		KeyValueList = append(KeyValueList, NamespaceKeyValue{Id: i, Name: pair.Name, Size: pair.Size, Access: pair.Access})
	}
	return KeyValueList
}

func (App *Application) BadRequestHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v %v %v", r.Method, r.URL.Path, http.StatusBadRequest)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 Bad Request"))
		return
	})
}
