// Package all registers the built-in secret providers. Import it for its side
// effects so provider.Get can resolve "bitwarden" and "1password".
package all

import (
	_ "github.com/s1ks1/bwenv/v3/internal/provider/bitwarden"
	_ "github.com/s1ks1/bwenv/v3/internal/provider/onepassword"
)
