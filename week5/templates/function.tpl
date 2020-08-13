func (srv *{{.Recv}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) {
    if !strings.Contains("{{.Method}}", r.Method) {
        body := response{Error: "bad method"}
        http.Error(w, body.String(), http.StatusNotAcceptable)
        return
    }
    {{if .Auth}}
    if r.Header.Get("X-Auth") != "100500" {
        body := response{Error: "unauthorized"}
        http.Error(w, body.String(), http.StatusForbidden)
        return
    }
    {{end -}}

    var values url.Values
    if r.Method == http.MethodGet {
        values = r.URL.Query()
    } else {
        r.ParseForm()
        values = r.Form
    }
    params := new({{.Param}})
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

    res, err := srv.{{.Name}}(context.Background(), *params)
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

