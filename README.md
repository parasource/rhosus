# Rhosus - distributed file system

#### This project is currently under active development, please don't use it yet

![rhosus](https://github.com/parasource/rhosus/blob/master/assets/logo_new.svg)

**Rhosus** is a fast multi-purpose distributed file system written in Go. It uses BoltDB for memory-based storage backup
and
uses etcd for service discovery and password-based authentication.

## What is Rhosus

It was originally developed as a scholar project at [Samara University](https://ssau.ru), but very soon I decided to
make something out of it.

### Rhosus architecture

In Rhosus there are two types of working units: Registry and Node. Node's only purpose is to store raw blocks on
machine. The main complexity is on Registry, which decides, where to store blocks, how to store it and so on.

## Getting started

### Installation

First you need to install etcd for service discovery.
Please follow steps on this [page](https://etcd.io/docs/v3.4/install/)

Once you installed etcd, you can now install **Rhosus**

```bash
$ go install github.com/parasource/rhosus
```

For very basic deployment you need at least one registry and one data node.

```bash
$ make build
...
$ bin/rhosusr
$ bin/rhosusd
```

### Starting up

The bare minimum env configuration:

**Registry**

```bash
API_ADDR=127.0.0.1:8001
CLUSTER_ADDR=127.0.0.1:8401
RHOSUS_PATH=/var/lib/rhosus
ETCD_ADRR=127.0.0.1:2379
```

**Data Node**

```bash
SERVICE_ADDR=127.0.0.1:4500
RHOSUS_PATH=/var/lib/rhosus
ETCD_ADRR=127.0.0.1:2379
```