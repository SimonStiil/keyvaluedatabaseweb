package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type Application struct {
	Config     ConfigType
	KVDBClient *Client
}

type KeyValue struct {
	Id    int
	Key   string
	Value string
	Lines int
}

func (App *Application) HealthActuator(w http.ResponseWriter, r *http.Request) {
	if App.Config.Prometheus.Enabled {
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method, "").Inc()
	}
	if !(r.URL.Path == "/system/health") {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	var reply Health
	if App.KVDBClient.GetHealth() {
		reply.Status = "UP"
		log.Printf("I %v %v %v", r.Method, r.URL.Path, http.StatusOK)
	} else {
		reply.Status = "DOWN"
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("I %v %v %v", r.Method, r.URL.Path, http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
	return
}

func (App *Application) RootController(w http.ResponseWriter, r *http.Request) {
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
		pair := rest.KVPairV1{Key: r.PostFormValue("key"),
			Value: r.PostFormValue("value")}
		var ok bool
		switch function {
		case "Create", "Update":
			ok = App.KVDBClient.Set(pair)
		case "Generate":
			ok = App.KVDBClient.Generate(pair.Key)
		case "Roll":
			ok = App.KVDBClient.Roll(pair.Key)
		case "Delete":
			ok = App.KVDBClient.Delete(pair.Key)
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
	kvlist := App.KVDBClient.GetList()
	KeyValueList := App.convertList(kvlist)
	w.WriteHeader(statuscode)
	// https://pkg.go.dev/html/template
	tmpl := template.Must(template.ParseFiles("index.html"))
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

func (App *Application) convertList(list []rest.KVPairV1) []KeyValue {
	var KeyValueList []KeyValue
	for i, pair := range list {
		KeyValueList = append(KeyValueList, KeyValue{Id: i, Key: pair.Key, Value: pair.Value, Lines: App.countRune(pair.Value, '\n')})
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
