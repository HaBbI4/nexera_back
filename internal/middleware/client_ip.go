package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIPResolver определяет реальный IP клиента с учетом прокси перед приложением (nginx).
//
// За прокси r.RemoteAddr — адрес самого прокси, а адрес клиента прокси дописывает в
// X-Forwarded-For. Но этот заголовок может прислать и сам клиент, подставив любой IP, поэтому
// верить ему можно только если запрос пришел от доверенного прокси — тот же принцип, что
// TrustProxies в Laravel.
type ClientIPResolver struct {
	trusted []netip.Prefix
}

// NewClientIPResolver создает ClientIPResolver. trustedProxies — адреса ("127.0.0.1")
// или подсети ("172.16.0.0/12"); пустой список — не доверять никому, IP всегда из соединения.
func NewClientIPResolver(trustedProxies []string) (*ClientIPResolver, error) {
	resolver := &ClientIPResolver{}

	for _, item := range trustedProxies {
		if strings.Contains(item, "/") {
			prefix, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy %q: %w", item, err)
			}
			resolver.trusted = append(resolver.trusted, prefix.Masked())
			continue
		}

		addr, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: %w", item, err)
		}
		addr = addr.Unmap()
		resolver.trusted = append(resolver.trusted, netip.PrefixFrom(addr, addr.BitLen()))
	}

	return resolver, nil
}

// ClientIP возвращает IP клиента. Если соединение пришло от доверенного прокси, X-Forwarded-For
// читается справа налево (каждый прокси дописывает адрес в конец), пропуская доверенные
// прокси: первый недоверенный адрес — клиент. Все, что левее, мог подделать сам клиент.
func (c *ClientIPResolver) ClientIP(r *http.Request) string {
	remote, ok := parseRemoteAddr(r.RemoteAddr)
	if !ok {
		return r.RemoteAddr
	}

	if !c.isTrusted(remote) {
		return remote.String()
	}

	var hops []string
	for _, header := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(header, ",")...)
	}

	client := remote
	for i := len(hops) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			// Мусор в заголовке — дальше влево верить нечему, клиент — последний
			// разобранный адрес.
			break
		}
		client = addr.Unmap()
		if !c.isTrusted(client) {
			break
		}
	}

	return client.String()
}

func (c *ClientIPResolver) isTrusted(addr netip.Addr) bool {
	for _, prefix := range c.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// parseRemoteAddr разбирает r.RemoteAddr ("ip:port"); IPv4-mapped IPv6 приводится к IPv4,
// чтобы "::ffff:127.0.0.1" совпадал с доверенным "127.0.0.1".
func parseRemoteAddr(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}
