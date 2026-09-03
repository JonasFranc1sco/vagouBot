package proxy

import (
	"net/http"
	"net/url"
	"sync/atomic"
)

// Rotator distribui requests entre uma lista de proxies.
type Rotator struct {
	proxies []*url.URL
	index   atomic.Int32
}

// New cria um rotator a partir de uma lista de URLs de proxy.
func New(proxyURLs []string) *Rotator {
	r := &Rotator{}
	for _, raw := range proxyURLs {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		r.proxies = append(r.proxies, u)
	}
	return r
}

// RoundRobin retorna o próximo proxy da lista.
// atomic.Int32 garante thread-safety.
func (r *Rotator) RoundRobin() *url.URL {
	if len(r.proxies) == 0 {
		return nil
	}
	idx := int(r.index.Add(1)-1) % len(r.proxies)
	return r.proxies[idx]
}

func (r *Rotator) Len() int {
	return len(r.proxies)
}

// ConfigureTransport aplica o proxy rotation no http.Transport.
func (r *Rotator) ConfigureTransport(transport *http.Transport) {
	if r.Len() == 0 {
		return
	}

	// Proxy é uma função, não um valor estático.
	// Go chama a função a cada request, então a rotação acontece automaticamente.
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return r.RoundRobin(), nil
	}
}
