package router

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/miekg/dns"
)

type HTTP struct {
	Client  *http.Client
	Handler http.HandlerFunc
	Port    int
	Router  *Router
	Scheme  string
}

func NewHTTP(r *Router, scheme string, port int) (*HTTP, error) {
	h := &HTTP{
		Client: &http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
			Transport:     defaultTransport(),
		},
		Router: r,
		Port:   port,
		Scheme: scheme,
	}

	return h, nil
}

func (h *HTTP) Serve() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", h.Port))
	if err != nil {
		return err
	}

	switch h.Scheme {
	case "https":
		ln = tls.NewListener(ln, &tls.Config{
			GetCertificate: h.generateCertificate,
		})
	}

	s := &http.Server{Handler: h}

	return s.Serve(ln)
}

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Handler != nil {
		h.Handler(w, r)
		return
	}

	h.ServeRequest(w, r)
}

func (h *HTTP) ServeRequest(w http.ResponseWriter, r *http.Request) {
	target := h.Router.TargetRandom(r.Host)
	if target == "" {
		http.Error(w, "no backends available", 502)
		return
	}

	if strings.ToLower(r.Header.Get("Connection")) == "upgrade" {
		h.serveWebsocket(w, r, target)
		return
	}

	fmt.Printf("ns=convox.router at=route host=%q method=%q path=%q\n", r.Host, r.Method, r.RequestURI)

	req, err := http.NewRequest(r.Method, fmt.Sprintf("%s%s", target, r.RequestURI), r.Body)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	req.Host = r.Host

	req.Header.Add("X-Forwarded-For", r.RemoteAddr)

	if v := req.Header.Get("X-Forwarded-Port"); v != "" {
		req.Header.Set("X-Forwarded-Port", v)
	} else {
		req.Header.Set("X-Forwarded-Port", strconv.Itoa(h.Port))
	}

	if v := req.Header.Get("X-Forwarded-Proto"); v != "" {
		req.Header.Set("X-Forwarded-Proto", v)
	} else {
		req.Header.Set("X-Forwarded-Proto", h.Scheme)
	}

	res, err := h.Client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}

	defer res.Body.Close()

	for k, vv := range res.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(res.StatusCode)

	if _, err := io.Copy(w, res.Body); err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
}

func (h *HTTP) generateCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return h.Router.generateCertificate(hello.ServerName)
}

var upgrader = websocket.Upgrader{ReadBufferSize: 10 * 1024, WriteBufferSize: 10 * 1024}

func websocketError(ws *websocket.Conn, err error) {
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
}

func websocketCopy(wsw, wsr *websocket.Conn) {
	defer wsw.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	for {
		t, r, err := wsr.NextReader()
		if cerr, ok := err.(*websocket.CloseError); ok {
			fmt.Println("reader")
			fmt.Printf("cerr = %+v\n", cerr)
			wsw.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(cerr.Code, cerr.Text))
			return
		}
		if err != nil {
			websocketError(wsw, err)
			return
		}

		w, err := wsw.NextWriter(t)
		if cerr, ok := err.(*websocket.CloseError); ok {
			fmt.Println("writer")
			fmt.Printf("cerr = %+v\n", cerr)
			// wsr.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(cerr.Code, cerr.Text))
			return
		}
		if err != nil {
			fmt.Println("writer")
			fmt.Printf("err = %+v\n", err)
			// websocketError(wsr, err)
			return
		}

		io.Copy(w, r)

		w.Close()
	}
}

func (h *HTTP) serveWebsocket(w http.ResponseWriter, r *http.Request, target string) {
	fmt.Printf("ns=convox.router at=websocket host=%q path=%q\n", r.Host, r.RequestURI)

	in, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(w, "error: %s\n", err)
		return
	}

	tu, err := url.Parse(target)
	if err != nil {
		websocketError(in, err)
		return
	}

	switch tu.Scheme {
	case "https":
		tu.Scheme = "wss"
	default:
		tu.Scheme = "ws"
	}

	tu.Path = r.URL.Path
	tu.RawQuery = r.URL.RawQuery

	r.Header.Set("Origin", target)
	r.Header.Add("X-Forwarded-For", r.RemoteAddr)

	if v := r.Header.Get("X-Forwarded-Port"); v != "" {
		r.Header.Set("X-Forwarded-Port", v)
	} else {
		r.Header.Set("X-Forwarded-Port", strconv.Itoa(h.Port))
	}

	if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
		r.Header.Set("X-Forwarded-Proto", v)
	} else {
		r.Header.Set("X-Forwarded-Proto", h.Scheme)
	}

	r.Header.Del("Connection")
	r.Header.Del("Sec-Websocket-Extensions")
	r.Header.Del("Sec-Websocket-Key")
	r.Header.Del("Sec-Websocket-Version")
	r.Header.Del("Upgrade")

	d := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}

	out, _, err := d.Dial(tu.String(), r.Header)
	if err != nil {
		websocketError(in, err)
		return
	}

	go websocketCopy(in, out)
	go websocketCopy(out, in)
}

func defaultDialer() *net.Dialer {
	return &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 10 * time.Second,
	}
}

func defaultTransport() *http.Transport {
	return &http.Transport{
		DialContext:           defaultDialer().DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

func targetBackendCount(target string) (int, error) {
	tu, err := url.Parse(target)
	if err != nil {
		return 0, fmt.Errorf("invalid target: %s", target)
	}

	m := &dns.Msg{}
	m.SetQuestion(fmt.Sprintf("_main._tcp.%s.", tu.Hostname()), dns.TypeSRV)

	cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return 0, err
	}
	if len(cfg.Servers) < 1 {
		return 0, fmt.Errorf("no dns servers found")
	}

	c := &dns.Client{}

	ma, _, err := c.Exchange(m, fmt.Sprintf("%s:%s", cfg.Servers[0], cfg.Port))
	if err != nil {
		return 0, err
	}

	for _, a := range ma.Answer {
		fmt.Printf("a = %+v\n", a)
	}

	return 1, nil
}
