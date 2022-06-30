package lib

import "net/http"

// InboundRequest is a struct that represents an inbound request
type InboundRequest struct {
	// W is the writer to write the response to
	W http.ResponseWriter
	// R is the reader to read the request from
	R *http.Request
	// EndpointLogic is the logic to handle the request
	EndpointLogic func() (interface{}, error)
	// ReqBody is the object where when applicable body of request has to be
	// unmarshalled
	ReqBody interface{}
	// StatusCode is the status code to return
	StatusCode int
}
