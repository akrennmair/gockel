include $(GOROOT)/src/Make.inc

TARG=gockel

GOFILES=gockel.go \
		twitterapi.go \
		model.go \
		findurls.go \
		ui.go


include $(GOROOT)/src/Make.cmd

gofmt:
	for f in $(GOFILES) ; do gofmt $$f > $$f.new ; mv $$f.new $$f ; done

.PHONY: gofmt
