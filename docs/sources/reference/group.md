page_title: group.yml reference
page_description: group.yml allows you to specify a group ofcontainers and their portable runtime properties for ease of container management and orchestration.
page_keywords: group, docker, yml, orchestration

# `group.yml` Reference

The `docker group` command allows users to manage groups
of containers with a common set of operations such as `create`, `list`,
`rm`, etc.  These groups can be created using the `docker up` command, which
will read a YAML file from the current working directory (called `group.yml`)
and use that to define a named group of containers, as well as portable runtime
properties that those containers will have.

The format of `group.yml` is as follows: There are two top-level keys, called
`name` and `containers`, which define the name of the group and the containers
that belong to it, respectively.

Each entry in the `containers` dictionary has a key which is the container name
, and a value which is a dictionary of runtime properties that container will
have.

Here is an example:

    name: my_django_app
    containers:
      db:
        image: postgres
      web:
        build: .
        command: python manage.py runserver 0.0.0.0:8000
        volumes:
          - .:/code
        ports:
          - "8000"

Container runtime keys loosely correspond to options passed to `docker run`
as flags.

> **Warning**
> Because one of the goals of Docker is interoperability across various
> environments, only portable runtime properties may be defined in
> `group.yml`.  Non-portable properties, such as binding container
> ports to a specific port on the host, are not allowed.

The following details the properties which can be defined in `group.yml`.

## build

_TYPE_: String

An alternative to the `image` key, the `build` key allows you to specify a
directory which contains a Dockerfile to use as an image.

Example:

    build: .

## cap_add

_TYPE_: Array

Allow kernel capabilities for the container.

Example:

    cap_add:
      - SYSADMIN
      - NET_ADMIN

## cap_drop

_TYPE_: Array

Disallow kernel capabilities for the container.

    cap_drop:
      - MKNOD
      - CHOWN

## command

_TYPE_: String

Specify the `CMD` for the container.  This command will be
run when the container starts or, if an `ENTRYPOINT` is also defined,
will be appended to the `ENTRYPOINT`.

Example:

    command: --default-flags --foo

## cpu_shares

Specify the relative number of CPU shares for the container.

## cpu_set
## devices
## entrypoint

_TYPE_: String

Specify an `ENTRYPOINT` for the container.

Example:

    entrypoint: python -m SimpleHTTPServer

## environment

_TYPE_: Array

Specify environment variables that will be populated in the container
when it is run.  The listed environment variables should be in the form
`KEY=VALUE`.

Example:

    environment:
      - DEV=yes
      - FOO=BAR
      - QUUX=PAAS

## image

_TYPE_: String

The image to use for the container.  Either the `image` key or the `build`
key _must_ be defined for a container in order for the group to be created
successfully.

Example:

    image: postgres:latest

## memory

Specify a memory limit for the container.

## ports

_TYPE_: Array

Specify ports the container will expose to other containers.

Example:

    ports:
      - "8000"
      - "6379"

## privileged

_TYPE_: Boolean

Specify whether or not the container should be run in privileged mode.

## tty
## user

_TYPE_: String

Specify a user to run the container as.

## volumes

_TYPE_: Array

Specify volumes the container will have.

Example:

    volumes:
      - /var/lib/postgresql
      - /var/log/foo

## volumes_from

_TYPE_: String

Specify a container to inherit volumes from.

    volumes_from: data_container

## working_dir

_TYPE_: String

Specify a working directory which the container's entry process will be run in.

Example:

    working_dir: /code
