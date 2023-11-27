package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

type Application struct {
	Config     ConfigType
	Content    []KeyValue
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
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	}
	if !(r.URL.Path == "/system/health") {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	reply := Health{Status: "UP"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
	return
}

func (App *Application) RootController(w http.ResponseWriter, r *http.Request) {
	requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			if App.Config.Debug {
				log.Printf("ParseForm: %v, %t\n", err, err)
			}
			App.BadRequestHandler().ServeHTTP(w, r)
		}
	}
	log.Printf("%v %v %v %+v", r.Method, r.URL.Path, 200, r.PostForm)
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, App.Content)
}

func (App *Application) BadRequestHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v %v %v", r.Method, r.URL.Path, 400)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 Bad Request"))
		return
	})
}
