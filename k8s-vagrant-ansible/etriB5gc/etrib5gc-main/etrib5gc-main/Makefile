controller:
	go build -o bin/controller mesh/controller/cmd/main.go 
gateway:
	go build -o bin/gateway mesh/gateway/cmd/main.go 
nsm:
	go build -o bin/nsm nfs/nsm/cmd/main.go	
amf:
	go build -o bin/amf nfs/amf/cmd/main.go	
damf:
	go build -o bin/damf nfs/damf/cmd/main.go	
udm:
	go build -o bin/udm nfs/udm/cmd/main.go	
ausf:
	go build -o bin/ausf nfs/ausf/cmd/main.go	
smf:
	go build -o bin/smf nfs/smf/cmd/main.go	
pcf:
	go build -o bin/pcf nfs/pcf/cmd/main.go	
pran:
	go build -o bin/pran nfs/pran/cmd/main.go	
upf:
	go build -o bin/upf nfs/upf/cmd/main.go	
sepp:
	go build -o bin/sepp nfs/sepp/cmd/main.go	
docker:
	cd docker; make
ue-gen:
	go build -o bin/ue-gen util/ue-gen/*.go

sbigen:
	go build -o bin/sbigen util/sbigen/*.go

oam:
	go build -o bin/oam-cli util/oam-cli/main.go	

clean:
	rm bin/*
docker-clean:
	cd docker; make clean

.PHONY: docker
.DEFAULT_GOAL := all
all: controller gateway nsm pran damf amf udm ausf smf pcf upf sepp ue-gen sbigen oam
