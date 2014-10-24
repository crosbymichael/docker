package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/docker/docker/api"
	"gopkg.in/yaml.v1"
)

var yamlData = []byte(`name: everything

containers:
  everything:
    image: busybox

    command: sleep infinity
    entrypoint: ["echo"]
    environment:
      - FOO=1

    ports:
      - "8001:8000"
    volumes:
      - /data
      - .:/code
    links:
      - redis

    user: root
    working_dir: /workdir
    tty: true

    memory: 1g
    cpu_shares: 100
    cpu_set: 0,1

    privileged: true
    cap_add:
      - NET_ADMIN
    cap_drop:
      - MKNOD
    devices:
      - /dev/sdc:/dev/xvdc:rwm
`)

var jsonData = []byte(`{
  "name": "everything",

  "containers": [
    {
      "Name": "everything",
      "Image": "busybox",

      "Cmd": ["/bin/sh", "-c", "sleep infinity"],
      "Entrypoint": ["echo"],
      "Env": [
        "FOO=1"
      ],

      "Ports": [
        {
          "Proto": "tcp",
          "Container": 8000,
          "Host": 8001
        }
      ],

      "Volumes": [
        {
          "Container": "/data",
          "Host": "",
          "Mode": "rw"
        },
        {
          "Container": "/code",
          "Host": "/pwd/on/host",
          "Mode": "rw"
        }
      ],

      "Links": [
        "redis"
      ],

      "User": "root",
      "WorkingDir": "/workdir",
      "Tty": true,

      "Memory": 1073741824,
      "CpuShares": 100,
      "Cpuset": "0,1",

      "Privileged": true,
      "CapAdd": ["NET_ADMIN"],
      "CapDrop": ["MKNOD"],
      "Devices": [
        {
          "PathOnHost": "/dev/sdc",
          "PathInContainer": "/dev/xvdc",
          "CgroupPermissions": "rwm"
        }
      ]
    }
  ]
}`)

var expected = &api.Group{
	Name: "everything",

	Containers: []*api.Container{
		&api.Container{
			Name:  "everything",
			Image: "busybox",

			Cmd:        []string{"/bin/sh", "-c", "sleep infinity"},
			Entrypoint: []string{"echo"},
			Env: []string{
				"FOO=1",
			},

			Ports: []*api.Port{
				&api.Port{
					Proto:     "tcp",
					Container: 8000,
					Host:      8001,
				},
			},

			Volumes: []*api.Volume{
				&api.Volume{
					Container: "/data",
					Host:      "",
					Mode:      "rw",
				},
				&api.Volume{
					Container: "/code",
					Host:      "/pwd/on/host",
					Mode:      "rw",
				},
			},

			Links: []string{
				"redis",
			},

			User:       "root",
			WorkingDir: "/workdir",
			Tty:        true,

			Memory:    1073741824,
			CpuShares: 100,
			Cpuset:    "0,1",

			Privileged: true,
			CapAdd:     []string{"NET_ADMIN"},
			CapDrop:    []string{"MKNOD"},
			Devices: []*api.Device{
				&api.Device{
					PathOnHost:        "/dev/sdc",
					PathInContainer:   "/dev/xvdc",
					CgroupPermissions: "rwm",
				},
			},
		},
	},
}

func TestYAML(t *testing.T) {
	var raw *GroupConfig
	if err := yaml.Unmarshal(yamlData, &raw); err != nil {
		t.Fatal(err)
	}

	fromYAML, err := preprocessGroupConfig(raw, &GroupContext{
		CurrentDirectory: "/pwd/on/host",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := DeepCompare(fromYAML, expected); err != nil {
		t.Fatal(err)
	}
}

func TestJSON(t *testing.T) {
	var fromJSON *api.Group
	if err := json.Unmarshal(jsonData, &fromJSON); err != nil {
		t.Fatal(err)
	}

	if err := DeepCompare(fromJSON, expected); err != nil {
		t.Fatal(err)
	}
}

// Copied + modified from https://golang.org/src/pkg/reflect/deepequal.go

type visit struct {
	a1  uintptr
	a2  uintptr
	typ reflect.Type
}

func deepValueCompare(v1, v2 reflect.Value, visited map[visit]bool, depth int) error {
	if v1.IsValid() && !v2.IsValid() {
		return fmt.Errorf("%#v is valid but %#v is not", v1, v2)
	}
	if !v1.IsValid() && v2.IsValid() {
		return fmt.Errorf("%#v is not valid but %#v is", v1, v2)
	}
	if v1.Type() != v2.Type() {
		return fmt.Errorf("types differ: %#v (%#v) != %#v (%#v)", v1, v1.Type(), v2, v2.Type())
	}
	if v1.Type() == reflect.TypeOf(time.Time{}) {
		t1 := v1.Interface().(time.Time)
		t2 := v2.Interface().(time.Time)

		if t1.Equal(t2) {
			return nil
		} else {
			return fmt.Errorf("%s != %s", t1.String(), t2.String())
		}
	}

	// if depth > 10 { panic("deepValueCompare") }  // for debugging
	hard := func(k reflect.Kind) bool {
		switch k {
		case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
			return true
		}
		return false
	}

	if v1.CanAddr() && v2.CanAddr() && hard(v1.Kind()) {
		addr1 := v1.UnsafeAddr()
		addr2 := v2.UnsafeAddr()
		if addr1 > addr2 {
			// Canonicalize order to reduce number of entries in visited.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are identical ...
		if addr1 == addr2 {
			return nil
		}

		// ... or already seen
		typ := v1.Type()
		v := visit{addr1, addr2, typ}
		if visited[v] {
			return nil
		}

		// Remember for later.
		visited[v] = true
	}

	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			if err := deepValueCompare(v1.Index(i), v2.Index(i), visited, depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			return fmt.Errorf("%#v != %#v", v1.Interface(), v2.Interface())
		}
		if v1.Len() != v2.Len() {
			return fmt.Errorf("lengths differ (%#v): %#v != %#v", v1.Kind(), v1.Len(), v2.Len())
		}
		if v1.Pointer() == v2.Pointer() {
			return nil
		}
		for i := 0; i < v1.Len(); i++ {
			if err := deepValueCompare(v1.Index(i), v2.Index(i), visited, depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			return fmt.Errorf("%#v != %#v", v1.Interface(), v2.Interface())
		}
		return deepValueCompare(v1.Elem(), v2.Elem(), visited, depth+1)
	case reflect.Ptr:
		return deepValueCompare(v1.Elem(), v2.Elem(), visited, depth+1)
	case reflect.Struct:
		for i, n := 0, v1.NumField(); i < n; i++ {
			if err := deepValueCompare(v1.Field(i), v2.Field(i), visited, depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return fmt.Errorf("%#v != %#v", v1.Interface(), v2.Interface())
		}
		if v1.Len() != v2.Len() {
			return fmt.Errorf("lengths differ (%#v): %#v != %#v", v1.Kind(), v1.Len(), v2.Len())
		}
		if v1.Pointer() == v2.Pointer() {
			return nil
		}
		for _, k := range v1.MapKeys() {
			if err := deepValueCompare(v1.MapIndex(k), v2.MapIndex(k), visited, depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Func:
		if v1.IsNil() && v2.IsNil() {
			return nil
		}
		// Can't do better than this:
		return fmt.Errorf("can't compare functions")
	default:
		// Normal equality suffices
		if v1.Interface() != v2.Interface() {
			return fmt.Errorf("%#v != %#v", v1.Interface(), v2.Interface())
		}

		return nil
	}
}

func DeepCompare(a1, a2 interface{}) error {
	if a1 == nil || a2 == nil {
		if a1 == nil && a2 == nil {
			return nil
		} else {
			return fmt.Errorf("%#v != %#v", a1, a2)
		}
	}
	v1 := reflect.ValueOf(a1)
	v2 := reflect.ValueOf(a2)
	if v1.Type() != v2.Type() {
		return fmt.Errorf("types differ: %#v (%#v) != %#v (%#v)", v1, v1.Type(), v2, v2.Type())
	}
	return deepValueCompare(v1, v2, make(map[visit]bool), 0)
}
