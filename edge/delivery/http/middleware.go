package http

import (
	"log"
	"net"
	"net/http"
)

// cidrFilter — middleware, ограничивающий доступ по IP-подсетям.
//
// Если список подсетей пуст — доступ разрешён всем (фильтр выключен).
// Иначе запрос пропускается только если IP клиента (RemoteAddr) входит
// хотя бы в одну из подсетей. Иначе — 403.
//
// Важно: используем именно RemoteAddr (реальный адрес TCP-соединения),
// а НЕ X-Forwarded-For — заголовок подделывается клиентом и не должен
// влиять на контроль доступа.
func cidrFilter(allowed []*net.IPNet, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := remoteIP(r)
		if ip == nil || !ipAllowed(ip, allowed) {
			log.Printf("http: доступ запрещён для %s (%s %s)", r.RemoteAddr, r.Method, r.URL.Path)
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// remoteIP извлекает IP из RemoteAddr (формат "host:port").
func remoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr может быть без порта (редко) — пробуем как есть.
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

// ipAllowed проверяет, входит ли ip хотя бы в одну из подсетей.
func ipAllowed(ip net.IP, allowed []*net.IPNet) bool {
	for _, n := range allowed {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// parseCIDRs разбирает строковые CIDR в []*net.IPNet.
// Некорректные записи пропускаются с логом (не роняем сервер из-за опечатки).
func parseCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			log.Printf("http: некорректный CIDR %q пропущен: %v", c, err)
			continue
		}
		out = append(out, n)
	}
	return out
}
