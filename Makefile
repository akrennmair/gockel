include $(GOROOT)/src/Make.inc

TARG=gockel

GOFILES=gockel.go \
		twitterapi.go \
		controller.go \
		model.go \
		ui.go

include $(GOROOT)/src/Make.cmd
