package v2

import (
	"net/http"
	"time"

	"github.com/emicklei/go-restful"
)

// Container is the standard response type for the ContainerResource when interacting with
// docker containers.
type Container struct {
	JSONResponse
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
	Args    []string  `json:"args"`
	Image   string    `json:"image"`
}

// NewContainerResource returns a new restulf.Resource for managing and interacting with containers.
func NewContainerResource(d Docker) *ContainerResource {
	var (
		c = &ContainerResource{
			docker: d,
		}
		ws = newWebService("/containers", "Container resource for interaction with Docker containers")
	)
	ws.Route(ws.GET("/").To(c.getContainers).
		Doc("fetch all containers created in Docker").
		Operation("getContainers").
		Returns(http.StatusInternalServerError, "unable to fetch containers", JSONError{}).
		Param(ws.QueryParameter("filter", "filter containers from the response").DataType("string").AllowMultiple(true)).
		Writes([]Container{}))

	ws.Route(ws.GET("/{id}").To(c.getContainer).
		Doc("fetch a specific container by id or name").
		Operation("getContainer").
		Param(ws.PathParameter("id", "id/name of the container").DataType("string")).
		Returns(http.StatusNotFound, "no container found", JSONError{}).
		Writes(Container{}))

	ws.Route(ws.DELETE("/{id}").To(c.deleteContainer).
		Doc("delete the specified container by id or name").
		Notes("Delete removes the container and optionally any volumes specified").
		Operation("deleteContainer").
		Param(ws.PathParameter("id", "id/name of the container").DataType("string")).
		Param(ws.QueryParameter("kill", "kill the container before deleting it").DataType("bool")).
		Param(ws.QueryParameter("volume", "remove any volumes owned by the container").DataType("bool")).
		Returns(http.StatusNotFound, "no container found", JSONError{}))

	c.ws = ws
	return c
}

// ContainerResource handles opersations on Docker containers
type ContainerResource struct {
	ws     *restful.WebService
	docker Docker
}

// Add adds the ContainerResource to the restful.Container for routing in the REST API.
func (r *ContainerResource) Add(c *restful.Container) {
	c.Add(r.ws)
}

func (r *ContainerResource) getContainers(request *restful.Request, response *restful.Response) {
	containers, err := r.docker.FetchContainers()
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.WriteEntity(newJSONError(err))
		return
	}
	response.WriteEntity(containers)
}

func (r *ContainerResource) getContainer(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	container, err := r.docker.FetchContainer(id)
	if err != nil {
		response.WriteHeader(http.StatusNotFound)
		response.WriteEntity(newJSONError(err))
		return
	}
	response.WriteEntity(container)
}

func (r *ContainerResource) deleteContainer(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("id")
	if err := r.docker.DeleteContainer(id); err != nil {
		if err == ErrContainerNotFound {
			response.WriteHeader(http.StatusNotFound)
		} else {
			response.WriteHeader(http.StatusInternalServerError)
		}
		response.WriteEntity(newJSONError(err))
	}
}
