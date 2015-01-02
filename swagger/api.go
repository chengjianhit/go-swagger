package swagger

import (
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/casualjim/go-swagger/swagger/errors"
	"github.com/casualjim/go-swagger/swagger/spec"
)

// NewAPI creates the default untyped API
func NewAPI(spec *spec.Document) *API {
	return &API{
		spec: spec,
		consumers: map[string]Consumer{
			"application/json": JSONConsumer(),
		},
		producers: map[string]Producer{
			"application/json": JSONProducer(),
		},
		// authHandlers: make(map[string]AuthHandler),
		operations: make(map[string]OperationHandler),
		ServeError: errors.ServeError,
		Models:     make(map[string]func() interface{}),
	}
}

// API represents an untyped mux for a swagger spec
type API struct {
	spec      *spec.Document
	consumers map[string]Consumer
	producers map[string]Producer
	// authHandlers map[string]AuthHandler
	operations map[string]OperationHandler
	ServeError func(http.ResponseWriter, *http.Request, error)
	Models     map[string]func() interface{}
}

// // RegisterAuth registers an auth handler in this api
// func (d *API) RegisterAuth(scheme string, handler AuthHandler) {
// 	d.authHandlers[strings.ToUpper(scheme)] = handler
// }

// RegisterConsumer registers a consumer for a media type.
func (d *API) RegisterConsumer(mediaType string, handler Consumer) {
	d.consumers[strings.ToLower(mediaType)] = handler
}

// RegisterProducer registers a producer for a media type
func (d *API) RegisterProducer(mediaType string, handler Producer) {
	d.producers[strings.ToLower(mediaType)] = handler
}

// RegisterOperation registers an operation handler for an operation name
func (d *API) RegisterOperation(operationID string, handler OperationHandler) {
	d.operations[operationID] = handler
}

// OperationHandlerFor returns the operation handler for the specified id if it can be found
func (d *API) OperationHandlerFor(operationID string) (OperationHandler, bool) {
	h, ok := d.operations[operationID]
	return h, ok
}

// ConsumersFor gets the consumers for the specified media types
func (d *API) ConsumersFor(mediaTypes []string) map[string]Consumer {
	result := make(map[string]Consumer)
	for _, mt := range mediaTypes {
		if consumer, ok := d.consumers[mt]; ok {
			result[mt] = consumer
		}
	}
	return result
}

// ProducersFor gets the producers for the specified media types
func (d *API) ProducersFor(mediaTypes []string) map[string]Producer {
	result := make(map[string]Producer)
	for _, mt := range mediaTypes {
		if producer, ok := d.producers[mt]; ok {
			result[mt] = producer
		}
	}
	return result
}

// Validate validates this API for any missing items
func (d *API) Validate() error {
	return d.validate()
}

// validateWith validates the registrations in this API against the provided spec analyzer
func (d *API) validate() error {
	var consumes []string
	for k := range d.consumers {
		consumes = append(consumes, k)
	}
	var produces []string
	for k := range d.producers {
		produces = append(produces, k)
	}
	// TODO: implement auth handlers later
	// var authHandlers []string
	// for k := range d.authHandlers {
	// 	authHandlers = append(authHandlers, k)
	// }

	var operations []string
	for k := range d.operations {
		operations = append(operations, k)
	}

	if err := d.verify("consumes", consumes, d.spec.RequiredConsumes()); err != nil {
		return err
	}
	if err := d.verify("produces", produces, d.spec.RequiredProduces()); err != nil {
		return err
	}
	// TODO: hook auth in later on
	// if err := d.verify("auth scheme", schemes, s.structMapKeys(s.authSchemes)); err != nil {
	// 	return err
	// }
	if err := d.verify("operation", operations, d.spec.OperationIDs()); err != nil {
		return err
	}
	return nil
}

func (d *API) verify(name string, registrations []string, expectations []string) error {
	expected := map[string]struct{}{}
	seen := map[string]struct{}{}

	for _, v := range expectations {
		expected[v] = struct{}{}
	}

	var unspecified []string
	for _, v := range registrations {
		seen[v] = struct{}{}
		if _, ok := expected[v]; !ok {
			unspecified = append(unspecified, v)
		}
	}

	for k := range seen {
		delete(expected, k)
	}

	var unregistered []string
	for k := range expected {
		unregistered = append(unregistered, k)
	}

	if len(unregistered) > 0 || len(unspecified) > 0 {
		return &APIVerificationFailed{
			Section:              name,
			MissingSpecification: unspecified,
			MissingRegistration:  unregistered,
		}
	}

	return nil
}

// File represents an uploaded file.
type File struct {
	Data   multipart.File
	Header *multipart.FileHeader
}

// OperationHandlerFunc an adapter for a function to the OperationHandler interface
type OperationHandlerFunc func(interface{}) (interface{}, error)

// Handle implements the operation handler interface
func (s OperationHandlerFunc) Handle(data interface{}) (interface{}, error) {
	return s(data)
}

// OperationHandler a handler for a swagger operation
type OperationHandler interface {
	Handle(interface{}) (interface{}, error)
}

// ConsumerFunc represents a function that can be used as a consumer
type ConsumerFunc func(io.Reader, interface{}) error

// Consume consumes the reader into the data parameter
func (fn ConsumerFunc) Consume(reader io.Reader, data interface{}) error {
	return fn(reader, data)
}

// Consumer implementations know how to bind the values on the provided interface to
// data provided by the request body
type Consumer interface {
	// Consume performs the binding of request values
	Consume(io.Reader, interface{}) error
}

// ProducerFunc represents a function that can be used as a producer
type ProducerFunc func(io.Writer, interface{}) error

// Produce produces the response for the provided data
func (f ProducerFunc) Produce(writer io.Writer, data interface{}) error {
	return f(writer, data)
}

// Producer implementations know how to turn the provided interface into a valid
// HTTP response
type Producer interface {
	// Produce writes to the http response
	Produce(io.Writer, interface{}) error
}

// // AuthHandler handles authentication for an API
// type AuthHandler interface {
// 	// Authenticate peforms the authentication
// 	Authenticate(*http.Request) interface{}
// }
