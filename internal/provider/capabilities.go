package provider

import "fmt"

// The As* helpers turn a core Provider into an optional capability, returning a
// clear error when the provider does not implement it. Callers depend on the
// capability they actually use instead of the full method set.

// AsAuthenticator returns the provider as an Authenticator.
func AsAuthenticator(p Provider) (Authenticator, error) {
	a, ok := p.(Authenticator)
	if !ok {
		return nil, fmt.Errorf("provider %q does not support authentication", p.Slug())
	}
	return a, nil
}

// AsFolderLister returns the provider as a FolderLister.
func AsFolderLister(p Provider) (FolderLister, error) {
	fl, ok := p.(FolderLister)
	if !ok {
		return nil, fmt.Errorf("provider %q does not expose folders", p.Slug())
	}
	return fl, nil
}

// AsSecretFetcher returns the provider as a SecretFetcher.
func AsSecretFetcher(p Provider) (SecretFetcher, error) {
	sf, ok := p.(SecretFetcher)
	if !ok {
		return nil, fmt.Errorf("provider %q cannot fetch secrets", p.Slug())
	}
	return sf, nil
}

// AsLocker returns the provider as a Locker.
func AsLocker(p Provider) (Locker, error) {
	l, ok := p.(Locker)
	if !ok {
		return nil, fmt.Errorf("provider %q cannot be locked", p.Slug())
	}
	return l, nil
}
