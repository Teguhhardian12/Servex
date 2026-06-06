.PHONY: build clean

CGO_CFLAGS  ?= -DSQLITE_ENABLE_FTS5
CGO_LDFLAGS ?= -lm

build:
	CGO_ENABLED=1 CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o servex .

clean:
	rm -f servex servex.db
