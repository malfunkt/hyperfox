FROM fedora:31

RUN dnf install -y \
  # Commong tools
  git \
  tar \
  flex \
  byacc \
  unzip \
  wget \
  make \
  file \
  python \
  # Linux x86 and x64
  gcc \
  gcc-c++ \
  libgcc.i686 \
  gcc-c++.i686 \
  glibc-devel \
  glibc-static \
  glibc-devel.i686 \
  glib2-static.i686 \
  libpcap.x86_64 \
  libpcap.i686 \
  libpcap-devel.x86_64 \
  libpcap-devel.i686 \
  # Windows x64
  mingw32-gcc.x86_64 \
  mingw64-gcc.x86_64 \
  mingw32-wpcap.noarch \
  mingw64-wpcap.noarch \
  && dnf clean packages

# For ARM cross compilation
RUN dnf install -y dnf-plugins-core && \
  dnf copr enable -y lantw44/arm-linux-gnueabi-toolchain && \
  dnf install -y arm-linux-gnueabi-{binutils,gcc,glibc} && \
  dnf clean packages

ENV GO_TARBALL=https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz

RUN curl --silent -L $GO_TARBALL | tar -xzf - -C /usr/local

ENV GOROOT /usr/local/go
ENV GOPATH /go
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

WORKDIR $GOPATH/src/github.com/malfunkt/hyperfox
COPY . .

RUN go mod vendor
