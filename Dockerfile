FROM golang

RUN apt-get update && apt-get upgrade -y && apt-get install -y git libgit2-dev \
cmake libssl-dev libssh2-1-dev

RUN cd /tmp && git clone https://github.com/libgit2/libgit2.git && cd libgit2 && mkdir build && cd build && cmake .. && \
cmake --build . && cmake --build . --target install && ldconfig

RUN go get github.com/esell/sir-forksalot

ENTRYPOINT $GOPATH/bin/sir-forksalot