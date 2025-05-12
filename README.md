# Ares

## Setup

### Dev

Install `go` and `deno`. Then run

`make install`

### Keycloak

Must turn on `Client authentication` and set the `Client secret` env value
found under the Credential tab in Keycloak to allow ares to do introspection
checks to keycloak, verifying the `accessToken` validity.

## Run

### Dev

Setup a live reload environment using

`make live`
