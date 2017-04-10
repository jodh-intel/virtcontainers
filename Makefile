all: binaries
	go build -gcflags "-N -l" $(go list ./... | grep -v /vendor/)
	cd hack/virtc && go build -gcflags "-N -l"

pause:
	make -C $@

binaries: pause

clean:
	make -C pause clean
	rm -f hack/virtc/virtc

install:
	install -D -m 755 pause/pause /usr/bin/pause

uninstall:
	rm -f /usr/bin/pause

.PHONY: \
	binaries \
	clean \
	install \
	pause \
	uninstall
