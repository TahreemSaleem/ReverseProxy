.PHONY: grproxy gserve

all: grproxy gserve

grproxy:
	$(MAKE) -C grproxy

gserve:
	$(MAKE) -C gserve
