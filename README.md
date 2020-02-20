# check_peers
Get peer list from https://adapools.org/peers and update in your node config yaml file.

# Install
Download from release and put it in the same location where your node config file is located.

# Build from source
1. Install Go: https://golang.org/doc/install
2. Download the source:
```
git clone https://github.com/kinqsley/check_peers.git
```
3. Build with Go:
```
go get
go build check_peers.go
```

# Usage
```bash
check_peers -config=node-config.yaml
```

# Note
Please make a copy of your node config file before running this program.

Brought to you by [WIRA] staking.wira.co
