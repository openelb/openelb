package lib

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Router is an interface which all rest Router must implement.
type Router interface {
	Register(httpRouter chi.Router)
}

// readRequestBody reads the request body and unmarshals it into the given object.
func readRequestBody(r *http.Request, bodyObj interface{}) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	switch bodyObj.(type) {
	case *[]byte:
		reflect.Indirect(reflect.ValueOf(bodyObj)).Set(reflect.ValueOf(body))
	default:
		err = json.Unmarshal(body, bodyObj)
	}
	return err
}

// ServeRequest handles an inbound request.
func ServeRequest(req InboundRequest) {
	if req.ReqBody != nil {
		if err := readRequestBody(req.R, req.ReqBody); err != nil {
			writeResponse(req.W, http.StatusBadRequest, err)
			return
		}
	}
	resp, err := req.EndpointLogic()
	if err != nil {
		switch {
		case errors.IsTimeout(err) || errors.IsServerTimeout(err):
			writeResponse(req.W, http.StatusGatewayTimeout, nil)
		case errors.IsNotFound(err):
			writeResponse(req.W, http.StatusNotFound, nil)
		case errors.IsAlreadyExists(err) || errors.IsConflict(err):
			writeResponse(req.W, http.StatusConflict, nil)
		case errors.IsBadRequest(err):
			writeResponse(req.W, http.StatusBadRequest, nil)
		case errors.IsTooManyRequests(err):
			writeResponse(req.W, http.StatusTooManyRequests, nil)
		case errors.IsNotAcceptable(err):
			writeResponse(req.W, http.StatusNotAcceptable, nil)
		default:
			writeResponse(req.W, http.StatusInternalServerError, nil)
		}
	}
	writeResponse(req.W, req.StatusCode, resp)
}

// writeResponse writes the response to the writer with status code and
// response body.
func writeResponse(w http.ResponseWriter, statusCode int, resp interface{}) error {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	if resp != nil {
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}
	}
	w.WriteHeader(statusCode)
	return nil
}
