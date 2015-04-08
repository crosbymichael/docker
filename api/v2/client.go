package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// NewClient returns a Docker API client for interacting with the REST API.
func NewClient(uri string) (*Client, error) {
	c := &http.Client{}
	return &Client{
		c:   c,
		uri: uri,
	}, nil
}

type Client struct {
	c   *http.Client
	uri string
}

var _ = (Docker)(&Client{})

// FetchContainer returns the container for the provided id or name.
func (c *Client) FetchContainer(id string) (*Container, error) {
	r, err := c.c.Get(fmt.Sprintf("%s/containers/%s", c.uri, id))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var out *Container
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchContainers returns all container's from docker.
func (c *Client) FetchContainers() ([]*Container, error) {
	r, err := c.c.Get(fmt.Sprintf("%s/containers", c.uri))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var out []*Container
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteContainer(id string) error {
	r, err := http.NewRequest("DELETE", fmt.Sprintf("%s/containers/%s", c.uri, id), nil)
	if err != nil {
		return err
	}
	response, err := c.c.Do(r)
	if err != nil {
		return err
	}
	if response.StatusCode == 404 {
		return ErrContainerNotFound
	}
	return nil

}
