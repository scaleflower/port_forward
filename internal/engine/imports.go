package engine

import (
	// Register listeners
	_ "github.com/go-gost/x/listener/tcp"
	_ "github.com/go-gost/x/listener/udp"

	// Register handlers
	_ "github.com/go-gost/x/handler/forward/local" // forward handler
	_ "github.com/go-gost/x/handler/forward/remote"
	_ "github.com/go-gost/x/handler/http"
	_ "github.com/go-gost/x/handler/socks/v5"
	_ "github.com/go-gost/x/handler/ss"

	// Register connectors (for chains)
	_ "github.com/go-gost/x/connector/http"
	_ "github.com/go-gost/x/connector/socks/v5"
	_ "github.com/go-gost/x/connector/ss"

	// Register dialers
	_ "github.com/go-gost/x/dialer/tcp"
	_ "github.com/go-gost/x/dialer/tls"
)
