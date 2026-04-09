FROM b5gc-base as upfbuild

WORKDIR ${GOPATH}/src/
RUN apt-get -y update \
    && apt-get -y install linux-headers-amd64 git gcc g++ cmake libmnl-dev autoconf libtool libyaml-dev

RUN git clone --recurse-submodules https://github.com/free5gc/free5gc.git \
 && cd free5gc \
 && make upf
 
RUN git clone https://github.com/free5gc/go-gtp5gnl.git && \
    cd "go-gtp5gnl/cmd/gogtp5g-tunnel" &&  go build -o "${GOPATH}/gtp5g-tunnel" .

RUN mkdir /free5gc \
    && cp ${GOPATH}/src/free5gc/bin/upf /free5gc/ \
    && cp ${GOPATH}/src/free5gc/config/upfcfg.yaml /free5gc/upf.yaml \
    && cp ${GOPATH}/gtp5g-tunnel /free5gc/ 


FROM bitnami/minideb:bullseye

LABEL description="Free5GC open source 5G Core Network" \
    version="Stage 3"

ENV DEBIAN_FRONTEND noninteractive
ARG DEBUG_TOOLS

# Set working dir
WORKDIR /free5gc
COPY ${KVER} /lib/modules/${KVER}

# Install debug tools ~ 100MB (if DEBUG_TOOLS is set to true)
RUN if [ "$DEBUG_TOOLS" = "true" ] ; then apt-get update && apt-get install -y vim strace net-tools iputils-ping curl netcat ; fi

# Install UPF dependencies
RUN apt-get update \
    && apt-get install -y libmnl0 libyaml-0-2 iproute2 iptables \
    && apt-get clean


RUN mkdir -p config/ log/

# Copy executable
COPY --from=upfbuild /free5gc/upf ./
COPY --from=upfbuild /free5gc/upf.yaml ./
COPY --from=upfbuild /free5gc/gtp5g-tunnel ./

#RUN cd gtp5g; make; make install
 
# Config files volume
#VOLUME [ "/free5gc/config" ]
