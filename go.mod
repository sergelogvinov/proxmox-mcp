module github.com/sergelogvinov/proxmox-mcp

go 1.26.6

replace github.com/sergelogvinov/go-proxmox-rest => ../proxmox/go-proxmox-rest

require (
	github.com/modelcontextprotocol/go-sdk v1.7.0
	github.com/pkg/errors v0.9.1
	github.com/sergelogvinov/go-proxmox-rest v0.0.0-20260906113314-d4aca9ca5453
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.12.1
	go.yaml.in/yaml/v3 v3.0.5
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	resty.dev/v3 v3.0.0-rc.3 // indirect
)
