FROM golang:latest AS builder
WORKDIR /go/src/github.com/parasource/rhosus/
COPY . .

#ARG SSH_KEY

## Authorize SSH Host
#RUN mkdir -p /root/.ssh && \
#    chmod 0700 /root/.ssh && \
#    ssh-keyscan github.com > /root/.ssh/known_hosts
#
##  Moving our ssh key
#RUN echo ${SSH_KEY} > /root/.ssh/id_rsa && \
#    chmod 600 /root/.ssh/id_rsa

RUN go mod download

RUN make build

# starting from scratch

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /go/src/github.com/parasource/rhosus .

CMD ["./bin/rhosusr"]