BINARY = rhosus
PROJECTNAME=$(shell basename "$(PWD)")

GOBASE=$(shell pwd)
GOPATH=$(GOBASE)/vendor:$(GOBASE):/home/azer/code/golang  #Вы можете удалить или изменить путь после двоеточия.
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

SOURCE_DIR = .

.PHONY : debug_mount

debug_mount:
	go build -o ../build/rhosus .