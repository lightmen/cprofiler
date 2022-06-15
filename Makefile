LDFLAGS="-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse --short HEAD` -X 'main.goversion=$(go version)'"

RACE=
ENV=
ifeq ($(ENV),dev)
	RACE=-race
endif

all: cprofiler

cprofiler: 
	go build $(RACE) -ldflags $(LDFLAGS)  -o ./output/bin/cprofiler  ./server/

clean:
	go clean ./cmd/...

