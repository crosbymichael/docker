package v2

import (
	"errors"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
)

var (
	ErrContainerNotFound = errors.New("container does not exist")
)

// https://google-styleguide.googlecode.com/svn/trunk/jsoncstyleguide.xml#Top-Level_Reserved_Property_Names
type JSONResponse struct {
	Version string `json:"version"`
}

type JSONError struct {
	JSONResponse
	ID      int
	Message string   `json:"message"`
	Values  []string `json:"values"`
	DocURL  string   `json:"docUrl"`
}

// docker is the daemon level interface that is a dependency of the API.
type Docker interface {
	FetchContainer(id string) (*Container, error)
	FetchContainers() ([]*Container, error)
	DeleteContainer(id string) error
}

// New returns an http.Handler for the v2 REST API.
func New(d Docker, swaggerDist string) http.Handler {
	var (
		c  = restful.NewContainer()
		cr = NewContainerResource(d)
	)
	// add the container resource to the main application container.
	cr.Add(c)
	config := swagger.Config{
		WebServices:     c.RegisteredWebServices(),
		ApiPath:         "/apidocs.json",
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: swaggerDist,
	}
	swagger.RegisterSwaggerService(config, c)
	return c
}

// newWebService returns an initialized WebService that consumes and prodouces JSON.
func newWebService(path, doc string) *restful.WebService {
	ws := &restful.WebService{}
	ws.Path(path).
		Doc(doc).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	return ws
}

func newJSONError(err error) *JSONError {
	return &JSONError{
		Message: err.Error(),
		DocURL:  "https://docs.docker.com/api",
	}
}
