TARG=gockel
GO_SRC=$(wildcard *.go)

all: $(TARG)

$(TARG): $(GO_SRC)
	go build -o $(TARG)

clean:
	$(RM) $(TARG)

.PHONY: clean
