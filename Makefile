include $(GOROOT)/src/Make.inc

TARG=gockel

GOFILES=gockel.go \
		twitterapi.go \
		model.go \
		shortenurl.go \
		ui.go


include $(GOROOT)/src/Make.cmd

gofmt:
	for f in $(GOFMT) ; do gofmt $$f > $$f.new ; mv $$f.new $$f ; done

.PHONY: gofmt
