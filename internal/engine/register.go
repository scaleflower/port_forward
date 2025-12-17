package engine

// Import gost protocol implementations
// This file registers all supported protocols with the gost registry

import (
	// Register connectors
	_ "github.com/go-gost/x/connector/direct"
	_ "github.com/go-gost/x/connector/forward"
	_ "github.com/go-gost/x/connector/http"
	_ "github.com/go-gost/x/connector/relay"
	_ "github.com/go-gost/x/connector/socks/v5"
	_ "github.com/go-gost/x/connector/ss"
	_ "github.com/go-gost/x/connector/tcp"

	// Register dialers
	_ "github.com/go-gost/x/dialer/direct"
	_ "github.com/go-gost/x/dialer/tcp"
	_ "github.com/go-gost/x/dialer/tls"
	_ "github.com/go-gost/x/dialer/udp"
	_ "github.com/go-gost/x/dialer/ws"

	// Register handlers
	_ "github.com/go-gost/x/handler/forward/local"
	_ "github.com/go-gost/x/handler/forward/remote"
	_ "github.com/go-gost/x/handler/http"
	_ "github.com/go-gost/x/handler/relay"
	_ "github.com/go-gost/x/handler/socks/v5"
	_ "github.com/go-gost/x/handler/ss"

	// Register listeners
	_ "github.com/go-gost/x/listener/tcp"
	_ "github.com/go-gost/x/listener/tls"
	_ "github.com/go-gost/x/listener/udp"
	_ "github.com/go-gost/x/listener/ws"
)
