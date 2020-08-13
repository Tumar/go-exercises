
	package main

	import (
		"context"
		"encoding/json"
		"fmt"
		"net/http"
		"net/url"
		"strconv"
		"strings"
	)

	type response struct {
		Error    string       `json:"error"`
		Response interface{} `json:"response,omitempty"`
	}

	func (r *response) String() string {
		data, _ := json.Marshal(r)
		return string(data)
	}
	func (t *ProfileParams) bind(q url.Values) error {
	t.Login = q.Get("login")
	return nil
}

func (t *ProfileParams) validate(q url.Values) error {
	if t.Login == "" {
	return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("login must me not empty"),
		}
	}
	return nil
}

func (t *CreateParams) bind(q url.Values) error {
	Age, err := strconv.Atoi(q.Get("age"))
	if err != nil {
		return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("age must be int"),
		}
	}
	t.Age = Age
	
	t.Login = q.Get("login")
	
	t.Name = q.Get("full_name")
	
	t.Status = q.Get("status")
	return nil
}

func (t *CreateParams) validate(q url.Values) error {
	if t.Login == "" {
	return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("login must me not empty"),
		}
	}
	
	if len(t.Login) < 10 {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("login len must be >= 10"),
		}
	}
		 
	
	if t.Status == "" {
	t.Status = "user"
	}
	
	if !strings.Contains("|user|moderator|admin|", "|"+t.Status+"|") {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("status must be one of [user, moderator, admin]"),
		} 
	}
	
	if t.Age < 0 {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("age must be >= 0"),
		}
	}
		 
	
	if t.Age > 128 {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("age must be <= 128"),
		}
	}	 
	return nil
}


	func (srv *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains("GET|POST", r.Method) {
			body := response{Error: "bad method"}
			http.Error(w, body.String(), http.StatusNotAcceptable)
			return
		}
		var values url.Values
		if r.Method == http.MethodGet {
			values = r.URL.Query()
		} else {
			r.ParseForm()
			values = r.Form
		}
		params := new(ProfileParams)
		err := params.bind(values)
		if err != nil {
			err := err.(ApiError)
			body := response{Error: err.Error()}
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
	
		err = params.validate(values)
		if err != nil {
			err := err.(ApiError)
			body := response{Error: err.Error()}
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
	
		res, err := srv.Profile(context.Background(), *params)
		if err != nil {
			body := response{Error: err.Error()}
			if err, ok := err.(ApiError); ok {
				http.Error(w, body.String(), err.HTTPStatus)
				return
			}
			
			http.Error(w, body.String(), http.StatusInternalServerError)
			return
		}
	
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		body := response{Error: "", Response: res}
		fmt.Fprintln(w, body.String())
	}
	
	func (srv *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains("POST", r.Method) {
			body := response{Error: "bad method"}
			http.Error(w, body.String(), http.StatusNotAcceptable)
			return
		}
		
		if r.Header.Get("X-Auth") != "100500" {
			body := response{Error: "unauthorized"}
			http.Error(w, body.String(), http.StatusForbidden)
			return
		}
		var values url.Values
		if r.Method == http.MethodGet {
			values = r.URL.Query()
		} else {
			r.ParseForm()
			values = r.Form
		}
		params := new(CreateParams)
		err := params.bind(values)
		if err != nil {
			err := err.(ApiError)
			body := response{Error: err.Error()}
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
	
		err = params.validate(values)
		if err != nil {
			err := err.(ApiError)
			body := response{Error: err.Error()}
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
	
		res, err := srv.Create(context.Background(), *params)
		if err != nil {
			body := response{Error: err.Error()}
			if err, ok := err.(ApiError); ok {
				http.Error(w, body.String(), err.HTTPStatus)
				return
			}
			
			http.Error(w, body.String(), http.StatusInternalServerError)
			return
		}
	
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		body := response{Error: "", Response: res}
		fmt.Fprintln(w, body.String())
	}
	func (t *OtherCreateParams) bind(q url.Values) error {
	t.Username = q.Get("username")
	
	t.Name = q.Get("account_name")
	
	t.Class = q.Get("class")
	
	Level, err := strconv.Atoi(q.Get("level"))
	if err != nil {
		return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("level must be int"),
		}
	}
	t.Level = Level
	return nil
}

func (t *OtherCreateParams) validate(q url.Values) error {
	if t.Username == "" {
	return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("username must me not empty"),
		}
	}
	
	if len(t.Username) < 3 {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("username len must be >= 3"),
		}
	}
		 
	
	if t.Class == "" {
	t.Class = "warrior"
	}
	
	if !strings.Contains("|warrior|sorcerer|rouge|", "|"+t.Class+"|") {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("class must be one of [warrior, sorcerer, rouge]"),
		} 
	}
	
	if t.Level < 1 {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("level must be >= 1"),
		}
	}
		 
	
	if t.Level > 50 {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("level must be <= 50"),
		}
	}	 
	return nil
}


	func (srv *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains("POST", r.Method) {
			body := response{Error: "bad method"}
			http.Error(w, body.String(), http.StatusNotAcceptable)
			return
		}
		
		if r.Header.Get("X-Auth") != "100500" {
			body := response{Error: "unauthorized"}
			http.Error(w, body.String(), http.StatusForbidden)
			return
		}
		var values url.Values
		if r.Method == http.MethodGet {
			values = r.URL.Query()
		} else {
			r.ParseForm()
			values = r.Form
		}
		params := new(OtherCreateParams)
		err := params.bind(values)
		if err != nil {
			err := err.(ApiError)
			body := response{Error: err.Error()}
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
	
		err = params.validate(values)
		if err != nil {
			err := err.(ApiError)
			body := response{Error: err.Error()}
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
	
		res, err := srv.Create(context.Background(), *params)
		if err != nil {
			body := response{Error: err.Error()}
			if err, ok := err.(ApiError); ok {
				http.Error(w, body.String(), err.HTTPStatus)
				return
			}
			
			http.Error(w, body.String(), http.StatusInternalServerError)
			return
		}
	
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		body := response{Error: "", Response: res}
		fmt.Fprintln(w, body.String())
	}
	
	func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		
		case "/user/profile": srv.handleProfile(w, r)
		
		case "/user/create": srv.handleCreate(w, r)
		default:
			body := response{Error: "unknown method"}
			http.Error(w, body.String(), http.StatusNotFound)
		}
	}
	
	func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		
		case "/user/create": srv.handleCreate(w, r)
		default:
			body := response{Error: "unknown method"}
			http.Error(w, body.String(), http.StatusNotFound)
		}
	}
	