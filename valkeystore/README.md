# valkeystore

A [Valkey](https://github.com/valkey-io/valkey-go) based session store for [SCS](https://github.com/alexedwards/scs).

## Setup

To use `valkeystore`, you need to set up a connection to a Valkey server or cluster. The connection can be initialized using `valkey.NewClient()` with the appropriate connection URL. Pass the client to `valkeystore.New()` to establish the session store.

## Example

```go
package main

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/alexedwards/scs/v2"
	"github.com/valkey-io/valkey-go"
	"github.com/your-repo/valkeystore"
)

var sessionManager *scs.SessionManager

func main() {
	// Use the SCS_VALKEY_DSN environment variable or fallback to a default.
	dsn := os.Getenv("SCS_VALKEY_DSN")
	if dsn == "" {
		dsn = "redis://localhost:6379/0"
	}

	// Initialize a Valkey client.
	client, err := valkey.NewClient(valkey.MustParseURL(dsn))
	if err != nil {
		panic(err)
	}

	// Initialize a new session manager and configure it to use valkeystore as the session store.
	sessionManager = scs.New()
	sessionManager.Store = valkeystore.New(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/put", putHandler)
	mux.HandleFunc("/get", getHandler)

	http.ListenAndServe(":4000", sessionManager.LoadAndSave(mux))
}

func putHandler(w http.ResponseWriter, r *http.Request) {
	sessionManager.Put(r.Context(), "message", "Hello from a Valkey session!")
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	msg := sessionManager.GetString(r.Context(), "message")
	io.WriteString(w, msg)
}
```

## Expired Session Cleanup

Valkey will automatically remove expired session keys based on the expiration settings. You do not need to implement any additional cleanup logic.

## Key Collisions

By default, keys are in the form `scs:session:<token>`. For example:

```
"scs:session:ZnirGwi2FiLwXeVlP5nD77IpfJZMVr6un9oZu2qtJrg"
```

Because the token is highly unique, key collisions are not a concern. However, if you're configuring *multiple session managers*, all using `valkeystore`, you may want the keys to have different prefixes depending on the session manager. You can achieve this using the `NewWithPrefix()` method:

```go
client, err := valkey.NewClient(valkey.MustParseURL("redis://localhost:6379/0"))
if err != nil {
    panic(err)
}

sessionManagerOne := scs.New()
sessionManagerOne.Store = valkeystore.NewWithPrefix(client, "scs:session:1:")

sessionManagerTwo := scs.New()
sessionManagerTwo.Store = valkeystore.NewWithPrefix(client, "scs:session:2:")
```

## Connecting to Valkey

The Valkey URL must start with `redis://`, `rediss://`, or `unix://`. Examples:

- **Single Node**: `redis://127.0.0.1:6379/0`
- **Cluster**: `redis://127.0.0.1:7001?addr=127.0.0.1:7002&addr=127.0.0.1:7003`
- **Sentinel**: `redis://127.0.0.1:26379/0?master_set=my_master`

For details, refer to the [Valkey documentation](https://github.com/valkey-io/valkey-go).

