# Root Makefile — delegates to docker/Makefile
# All targets run inside Docker; no local Go installation required.
#
# See docker/README.md for details.

.PHONY: all build build-all test vet tidy clean shell help

all build build-all test vet tidy clean shell help:
	$(MAKE) -f docker/Makefile $@
