# Rhosus - distributed file system

#### This project is currently under active development

![rhosus](https://github.com/parasource/rhosus/blob/master/assets/logo_new.svg)

**Rhosus** is a multi-purpose super-fast distributed file system (DFS) written in Go. It is build on top of BoltDB and
uses etcd for service discovery.

## What is Rhosus

### Rhosus architecture

In Rhosus, like in Hadoop, there are two types of working units: Registry and Node. Node's only purpose is to store raw
data on machine. The main complexity is on Registry, which decides, where to store data, how to store it and so on. 