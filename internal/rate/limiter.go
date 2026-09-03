package rate

import (
	"context"

	"golang.org/x/time/rate"
)

// Limiter controla a velocidade dos requests
type Limiter struct {
	limiter *rate.Limiter
}

// New cria um limiter com requests por segundo e burst.
func New(RequestsPerSecond float64, burst int) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(RequestsPerSecond), burst),
	}
}

// Wait bloqueia até que um token esteja disponível.
func (l *Limiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

// Allow verifica sem bloquear se um request pode ser feito agora.
func (l *Limiter) Allow() bool {
	return l.limiter.Allow()
}
