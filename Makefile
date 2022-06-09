REVISION=$(shell git rev-parse HEAD)
VERSION=
LDFLAGS="-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git describe --long --dirty --abbrev=14` -X 'main.goversion=$(go version)'"

RACE=
ENV=
ifeq ($(ENV),dev)
	RACE=-race
endif

# GOBIN=$(GOPATH)/bin
# PATH:=$(GOBIN):$(PATH)


all: cprofiler

cprofiler: 
	# go build $(RACE) -o ./output/bin/ -ldflags $(LDFLAGS) --tags $(ENV) ./server/
	go build $(RACE) -o ./output/bin/cprofiler ./server/

install:
	cp ./output/bin/* /data/servers/bin/

cfg:
	cp ./configs/cprofiler.yml /data/servers/conf/

clean:
	go clean ./cmd/...

