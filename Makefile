NAME=liveness-wrapper
DESTDIR=tmp
COMMIT=$(shell git rev-list --abbrev-commit -1 HEAD 2>/dev/null || echo HEAD)
VERSION=$(shell git describe --match "v*" --abbrev=0 2>/dev/null || echo v0.0.0)

all: liveness-wrapper


liveness-wrapper: *.go */*.go
	go build -ldflags="-s -w -X github.com/gandalfmagic/liveness-wrapper/cmd.commit=${COMMIT} -X github.com/gandalfmagic/liveness-wrapper/cmd.version=${VERSION}" .


bin/linux-amd64/liveness-wrapper: *.go */*.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/linux-amd64/liveness-wrapper -ldflags="-extldflags=-static -s -w -X github.com/gandalfmagic/liveness-wrapper/cmd.commit=${COMMIT} -X github.com/gandalfmagic/liveness-wrapper/cmd.version=${VERSION}" .


bin/linux-arm/liveness-wrapper: *.go */*.go
	CGO_ENABLED=0 GOARCH=arm GOOS=linux go build -o bin/linux-arm/liveness-wrapper -ldflags='-extldflags=-static -s -w -X github.com/gandalfmagic/liveness-wrapper/cmd.commit=${COMMIT} -X github.com/gandalfmagic/liveness-wrapper/cmd.version=${VERSION}' .


bin/linux-arm64/liveness-wrapper: *.go */*.go
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -o bin/linux-arm64/liveness-wrapper -ldflags='-extldflags=-static -s -w -X github.com/gandalfmagic/liveness-wrapper/cmd.commit=${COMMIT} -X github.com/gandalfmagic/liveness-wrapper/cmd.version=${VERSION}' .


.PHONY: install
install: liveness-wrapper
	install -d $(DESTDIR)/usr/bin
	install liveness-wrapper $(DESTDIR)/usr/bin/liveness-wrapper


$(NAME)-$(VERSION)-amd64.tar.gz: bin/linux-amd64/liveness-wrapper
	tar -cz -C bin/linux-amd64 liveness-wrapper > $(NAME)-$(VERSION)-amd64.tar.gz


$(NAME)-$(VERSION)-arm.tar.gz: bin/linux-arm/liveness-wrapper
	tar -cz -C bin/linux-arm liveness-wrapper > $(NAME)-$(VERSION)-arm.tar.gz


$(NAME)-$(VERSION)-arm64.tar.gz: bin/linux-arm64/liveness-wrapper
	tar -cz -C bin/linux-arm64 liveness-wrapper > $(NAME)-$(VERSION)-arm64.tar.gz


.PHONY: package
package: $(NAME)-$(VERSION)-amd64.tar.gz $(NAME)-$(VERSION)-arm.tar.gz $(NAME)-$(VERSION)-arm64.tar.gz


clean:
	rm -f liveness-wrapper
	rm -f $(NAME)-$(VERSION)-amd64.tar.gz
	rm -f $(NAME)-$(VERSION)-arm.tar.gz
	rm -f $(NAME)-$(VERSION)-arm64.tar.gz
	rm -rf bin/