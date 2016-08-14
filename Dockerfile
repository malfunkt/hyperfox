FROM fedora:23

RUN dnf install -y gcc gcc-c++ libgcc.i686 gcc-c++.i686 && dnf clean packages

RUN dnf install -y glibc-devel glibc-static && dnf clean packages
RUN dnf install -y glibc-devel.i686 glib2-static.i686 && dnf clean packages

# Requisites for ARM
# ARM EABI toolchain must be grabbed from an contributor repository, such as:
# https://copr.fedoraproject.org/coprs/lantw44/arm-linux-gnueabi-toolchain/
RUN dnf install -y 'dnf-command(config-manager)' && dnf clean packages
RUN rpm --import https://copr-be.cloud.fedoraproject.org/results/lantw44/arm-linux-gnueabi-toolchain/pubkey.gpg && \
	dnf config-manager --add-repo=https://copr.fedoraproject.org/coprs/lantw44/arm-linux-gnueabi-toolchain/repo/fedora-23/lantw44-arm-linux-gnueabi-toolchain-fedora-23.repo && \
	dnf install -y arm-linux-gnueabi-gcc arm-linux-gnueabi-binutils arm-linux-gnueabi-glibc && \
	dnf clean packages

RUN dnf install -y mingw32-gcc.x86_64 mingw64-gcc.x86_64 && dnf clean packages

RUN dnf install -y tar git mercurial && dnf clean packages

RUN curl 'https://storage.googleapis.com/golang/go1.6.3.linux-amd64.tar.gz' | tar -xvzf - -C /usr/local

RUN mkdir -p /app 

ENV GOROOT /usr/local/go
ENV GOPATH /app
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin
RUN go get -u github.com/kardianos/govendor

RUN mkdir -p /app/src/github.com/xiam/hyperfox

WORKDIR /app/src/github.com/xiam/hyperfox
