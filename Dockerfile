FROM golang:1.15 as builder
ADD . /go/src/github.com/pg2000/codecommit-pr-resource
WORKDIR /go/src/github.com/pg2000/codecommit-pr-resource
RUN curl -sL https://taskfile.dev/install.sh | sh
RUN ./bin/task build

FROM alpine:3.12 as resource
COPY --from=builder /go/src/github.com/pg2000/codecommit-pr-resource/build /opt/resource
RUN apk add --update --no-cache \
    git \
    openssh \
    python3 \
    py3-pip \
    && chmod +x /opt/resource/*

RUN pip install git-remote-codecommit
COPY scripts/askpass.sh /usr/local/bin/askpass.sh
ADD scripts/install_git_crypt.sh install_git_crypt.sh
RUN ./install_git_crypt.sh && rm ./install_git_crypt.sh

FROM resource
LABEL MAINTAINER=pg2000
