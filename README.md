# Rhosus - distributed file system

#### This project is currently under active development, please don't use it yet

![rhosus](https://github.com/parasource/rhosus/blob/master/assets/logo_new.svg)

**Rhosus** is a multi-purpose super-fast distributed file system (DFS) written in Go. It is build on top of BoltDB and
uses etcd for service discovery.

## What is Rhosus

It was originally developed as a scholar project at [Samara University](https://ssau.ru), but very soon I decided to
make something out of it.

### Rhosus architecture

In Rhosus there are two types of working units: Registry and Node. Node's only purpose is to store raw data on machine.
The main complexity is on Registry, which decides, where to store data, how to store it and so on. 