include $(GOROOT)/src/Make.inc

TARG=gockel

GOFILES=gockel.go \
		twitterapi.go \
		model.go \
		ui.go

include $(GOROOT)/src/Make.cmd
