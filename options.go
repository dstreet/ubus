package ubus

type Option func(b *Bus)

func WithTransport(t MessageTransport) Option {
	return func(b *Bus) {
		b.transport = t
	}
}

func WithMatcher(m EventMatcher) Option {
	return func(b *Bus) {
		b.matcher = m
	}
}
