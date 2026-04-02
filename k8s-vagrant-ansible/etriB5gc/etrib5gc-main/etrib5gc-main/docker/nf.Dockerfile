#Dockerfile for b5gc-base
FROM golang:1.23.3-bullseye as b5gc-base

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update \
    && apt-get -y install gcc cmake autoconf libtool pkg-config libmnl-dev libyaml-dev apt-transport-https ca-certificates git curl
#    && curl -sL https://deb.nodesource.com/setup_14.x | bash - \
#    && curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
#    && echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list \
#    && apt-get update \
#    && apt-get install -y nodejs yarn

# Clean apt cache
RUN apt-get clean

#RUN mkdir /b5gc-mod

WORKDIR /b5gc-mod

RUN mkdir -p $GOPATH/src/b5gc

COPY b5gc/go.mod $GOPATH/src/b5gc
COPY b5gc/go.sum $GOPATH/src/b5gc

RUN cd $GOPATH/src/b5gc \
    && go mod download 

RUN cp -r $GOPATH/pkg .

ENTRYPOINT /bin/sh -c bash


#Dockerfile for an NF build
FROM b5gc-base as nfbuild 

LABEL maintainer="tqtung@etri.re.kr"

ENV DEBIAN_FRONTEND noninteractive

ARG B5GC_MODULE

WORKDIR /b5gc

# Get B5gc
COPY b5gc $GOPATH/src/b5gc

# Copy modules
COPY --from=b5gc-base /b5gc-mod/pkg $GOPATH/

RUN cd $GOPATH/src/b5gc \
    && make ${B5GC_MODULE} \
    && cp bin/${B5GC_MODULE} /b5gc/ \
    && cp config/${B5GC_MODULE}.json /b5gc/
#RUN if [ ${B5GC_MODULE} = controller ]; then cp ${GOPATH}/src/b5gc/config/upmf-params.json /${GOPATH}/src/b5gc/config/nssf-params.json /b5gc/;fi


#Dockerfile for an NF
FROM busybox

WORKDIR /b5gc

ARG B5GC_MODULE

# Copy executables
COPY --from=nfbuild /b5gc/*.json ./
COPY --from=nfbuild /b5gc/${B5GC_MODULE} ./

CMD ["/b5gc/${B5GC_MODULE}", "-c", "${B5GC_MODULE}.json"] 

EXPOSE 8878
