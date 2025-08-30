#!/bin/sh
set -e

go env -w GOINSECURE=gitlab.mba.corp
go env -w GONOPROXY=gitlab.mba.corp
go env -w GOPRIVATE=gitlab.mba.corp

git config --global http."https://gitlab.mba.corp/".sslVerify false

git config --global --add safe.directory /workspaces/app

go mod tidy
