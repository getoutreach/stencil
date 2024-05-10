APP := stencil
OSS := true
_ := $(shell ./scripts/devbase.sh) 

include .bootstrap/root/Makefile

## <<Stencil::Block(targets)>>
post-stencil::
	./scripts/shell-wrapper.sh catalog-sync.sh
	make fmt
	yarn upgrade
## <</Stencil::Block>>
