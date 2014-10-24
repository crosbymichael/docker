package daemon

import (
	"testing"

	"github.com/docker/docker/api"
)

func TestSort2Containers(t *testing.T) {
	containers := []*api.Container{
		&api.Container{
			Name:  "web",
			Links: []string{"db"},
		},
		&api.Container{
			Name: "db",
		},
	}

	sorted, err := sortContainers(containers)
	if err != nil {
		t.Fatal(err)
	}

	assertNames(t, sorted, []string{"db", "web"})
}

func TestSort3Containers(t *testing.T) {
	containers := []*api.Container{
		&api.Container{
			Name:  "web",
			Links: []string{"db"},
		},
		&api.Container{
			Name: "data",
		},
		&api.Container{
			Name:  "db",
			Links: []string{"data"},
		},
	}

	sorted, err := sortContainers(containers)
	if err != nil {
		t.Fatal(err)
	}

	assertNames(t, sorted, []string{"data", "db", "web"})
}

func TestSelfDependency(t *testing.T) {
	containers := []*api.Container{
		&api.Container{
			Name:  "foo",
			Links: []string{"foo"},
		},
	}

	_, err := sortContainers(containers)
	if err == nil {
		t.Fatalf("expected an error")
	}
}

func TestCircularDependency(t *testing.T) {
	containers := []*api.Container{
		&api.Container{
			Name:  "foo",
			Links: []string{"bar"},
		},
		&api.Container{
			Name:  "bar",
			Links: []string{"baz"},
		},
		&api.Container{
			Name:  "baz",
			Links: []string{"foo"},
		},
	}

	_, err := sortContainers(containers)
	if err == nil {
		t.Fatalf("expected an error")
	}
}

func assertNames(t *testing.T, containers []*api.Container, expected []string) {
	if len(containers) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(containers))
	}

	actual := make([]string, len(containers))
	for idx, c := range containers {
		actual[idx] = c.Name
	}

	for idx, a := range actual {
		if expected[idx] != a {
			t.Fatalf("expected %#v, got %#v", expected, actual)
		}
	}
}
