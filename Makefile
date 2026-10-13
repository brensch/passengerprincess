# Build and ship ppp to the schlift.com VM (ARM64, Oracle Linux).
#   make build     cross-build deploy/out/ (API binary + frontend) in Docker from
#                  the committed code (git HEAD), so only commits get deployed
#   make deploy    build, copy to the VM, restart the service
#   make setup-vm  one-time VM setup (user, systemd unit, Caddy site, API key)
VM ?= opc@146.235.203.8

.PHONY: build deploy setup-vm
build:
	rm -rf deploy/out
	git archive --format=tar HEAD | DOCKER_BUILDKIT=1 docker build -f deploy/Dockerfile --target export -o deploy/out -

deploy: build
	rsync -rlt --delete --chmod=D755,F644 deploy/out/frontend/ $(VM):/tmp/ppp-frontend/
	scp deploy/out/ppp $(VM):/tmp/ppp
	ssh $(VM) 'sudo install -m 755 /tmp/ppp /opt/ppp/ppp && sudo rm -rf /opt/ppp/frontend && sudo mv /tmp/ppp-frontend /opt/ppp/frontend && sudo chown -R root:root /opt/ppp/frontend && rm /tmp/ppp && sudo systemctl restart ppp && sleep 2 && systemctl is-active ppp'

setup-vm:
	scp deploy/ppp.service deploy/ppp.caddyfile $(VM):/tmp/
	grep '^MAPS_API_KEY=' .env | ssh $(VM) 'sudo install -m 600 /dev/stdin /etc/ppp.env'
	ssh $(VM) 'sudo sh -s' < deploy/setup-vm.sh
