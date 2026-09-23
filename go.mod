module github.com/sergelogvinov/proxmox-mcp

go 1.27.1

// replace github.com/sergelogvinov/go-proxmox-rest => ../go-proxmox-rest
// replace github.com/sergelogvinov/go-proxmox-pool => ../go-proxmox-pool

require (
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/sergelogvinov/go-proxmox-pool v0.0.0-20260923034137-60822ad176fa
	github.com/sergelogvinov/go-proxmox-rest v0.0.0-20260922140521-cb1976ec06d4
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.12.1
	go.yaml.in/yaml/v3 v3.0.5
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	resty.dev/v3 v3.0.0-rc.4 // indirect
)
