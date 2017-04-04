LDFLAGS ?= "-w -s"

build:
	go build -ldflags $(LDFLAGS) -o cf-postgresql-broker
.PHONY: build

push: build
	cf push

clean:
	rm -rf cf-postgresql-broker
