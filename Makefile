node: export SECURITY_MODE=cookie
node: export KRATOS_BROWSER_URL=http://127.0.0.1:4433/
node: export KRATOS_PUBLIC_URL=http://127.0.0.1:4433/
node: export KRATOS_ADMIN_URL=http://127.0.0.1:4434/
node: export PORT=4455

node:
	npm install https://github.com/ory/kratos-selfservice-ui-node/ ts-node-dev
	npm explore kratos-selfservice-ui-node -- npm run start

kratos:
	bash <(curl https://raw.githubusercontent.com/ory/kratos/v0.5.4-alpha.1/install.sh) -b . v0.5.4-alpha.1
	./kratos -c .kratos.yml serve --dev

all: kratos node
	protoc --go_out=. -I loggy --go-grpc_out=requireUnimplementedServers=false:. loggy/loggy.proto 
	mv github.com/tuxcanfly/loggy/loggy/loggy_grpc.pb.go loggy/
	mv github.com/tuxcanfly/loggy/loggy/loggy.pb.go loggy/
	go build -o loggy.exe ./cmd/loggy
	rm -rf github.com

clean:
	rm -rf github.com loggy/loggy.pb.go loggy/loggy_grpc.pb.go *.exe test.db logs loggy.index

.PHONY: node clean
