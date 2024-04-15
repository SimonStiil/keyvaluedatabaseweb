package main

import (
	"math/rand"
	"net/http"
	"strings"
)

type RequestParameters struct {
	Method     string
	Api        string
	Namespace  string
	orgRequest *http.Request
	RequestIP  string
	ID         int
}

func GetRequestParameters(r *http.Request) *RequestParameters {
	slashSeperated := strings.Split(r.URL.Path[1:], "/")
	req := &RequestParameters{Method: r.Method, orgRequest: r, ID: RandomID()}
	if len(slashSeperated) > 0 {
		req.Api = slashSeperated[0]
	}
	if len(slashSeperated) > 1 {
		req.Namespace = slashSeperated[1]
	}
	return req
}
func RandomID() int {
	return rand.Intn(9999)
}
