USERHOST=root@anpr
CPATH=$$HOME/.ssh/ctl-%L-%r@%h:%p

build:
	go build

debug-build:
	go build -race


run: debug-build
	killall server || true
	./server --root=./test/site/static --config-file=./tools/config.yaml

format:
	go fmt

deploy: build
	ssh -nNf -M -S "$(CPATH)" $(USERHOST)
	rsync --progress -avz -e "ssh -S $(CPATH)" --delete --exclude=creds --exclude=sslcache --exclude=.git --exclude=tmp ./ $(USERHOST):~dashboard/server/
	ssh -S "$(CPATH)" $(USERHOST) chmod u+rX,go= -R ~dashboard/server/
	ssh -S "$(CPATH)" $(USERHOST) chmod u+wX -R ~dashboard/server/creds/
	ssh -S "$(CPATH)" $(USERHOST) chown dashboard -R ~dashboard/server/
	ssh -S "$(CPATH)" $(USERHOST) setcap 'cap_net_bind_service=+ep' ~dashboard/server/server
	ssh -O exit -S "$(CPATH)" $(USERHOST)

