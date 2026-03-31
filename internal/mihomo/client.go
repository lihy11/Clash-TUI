package mihomo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const requestTimeout = 6 * time.Second

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	secret     string

	wsMu    sync.Mutex
	logConn *websocket.Conn
}

func NewClient(endpoint, secret string) (*Client, error) {
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: u,
		httpClient: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				Proxy: nil,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		secret: secret,
	}, nil
}

func (c *Client) authHeader(req *http.Request) {
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, reqBody any, out any) error {
	ref := *c.baseURL
	ref.Path = path
	ref.RawQuery = query.Encode()

	var bodyReader *bytes.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, ref.String(), bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %s %s failed: %s", method, path, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) GetVersion(ctx context.Context) (VersionResponse, error) {
	var out VersionResponse
	err := c.doJSON(ctx, http.MethodGet, "/version", nil, nil, &out)
	return out, err
}

func (c *Client) GetConfig(ctx context.Context) (ConfigResponse, error) {
	var out ConfigResponse
	err := c.doJSON(ctx, http.MethodGet, "/configs", nil, nil, &out)
	return out, err
}

func (c *Client) SetMode(ctx context.Context, mode string) error {
	return c.doJSON(ctx, http.MethodPatch, "/configs", nil, UpdateConfigRequest{Mode: mode}, nil)
}

func (c *Client) SetSystemProxy(ctx context.Context, enable bool) error {
	return c.doJSON(ctx, http.MethodPatch, "/configs", nil, UpdateConfigRequest{
		SystemProxy: &enable,
	}, nil)
}

func (c *Client) SetTun(ctx context.Context, enable bool) error {
	tun := TunConfig{Enable: enable}
	return c.doJSON(ctx, http.MethodPatch, "/configs", nil, UpdateConfigRequest{
		Tun: &tun,
	}, nil)
}

func (c *Client) GetProxies(ctx context.Context) (ProxiesResponse, error) {
	var out ProxiesResponse
	err := c.doJSON(ctx, http.MethodGet, "/proxies", nil, nil, &out)
	return out, err
}

func (c *Client) SelectProxy(ctx context.Context, group, name string) error {
	path := "/proxies/" + url.PathEscape(group)
	return c.doJSON(ctx, http.MethodPut, path, nil, SelectProxyRequest{Name: name}, nil)
}

func (c *Client) TestProxyDelay(ctx context.Context, proxyName string, timeoutMS int) (DelayResponse, error) {
	path := "/proxies/" + url.PathEscape(proxyName) + "/delay"
	query := url.Values{
		"url":     []string{"https://www.gstatic.com/generate_204"},
		"timeout": []string{fmt.Sprintf("%d", timeoutMS)},
	}
	var out DelayResponse
	err := c.doJSON(ctx, http.MethodGet, path, query, nil, &out)
	return out, err
}

func (c *Client) GetRules(ctx context.Context) (RulesResponse, error) {
	var out RulesResponse
	err := c.doJSON(ctx, http.MethodGet, "/rules", nil, nil, &out)
	return out, err
}

func (c *Client) GetConnections(ctx context.Context) (ConnectionsResponse, error) {
	var out ConnectionsResponse
	err := c.doJSON(ctx, http.MethodGet, "/connections", nil, nil, &out)
	return out, err
}

func (c *Client) CloseConnection(ctx context.Context, id string) error {
	path := "/connections/" + url.PathEscape(id)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *Client) CloseAllConnections(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodDelete, "/connections", nil, nil, nil)
}

func (c *Client) GetProxyProviders(ctx context.Context) (ProvidersResponse, error) {
	var out ProvidersResponse
	err := c.doJSON(ctx, http.MethodGet, "/providers/proxies", nil, nil, &out)
	return out, err
}

func (c *Client) UpdateProxyProvider(ctx context.Context, name string) error {
	path := "/providers/proxies/" + url.PathEscape(name)
	return c.doJSON(ctx, http.MethodPut, path, nil, nil, nil)
}

func (c *Client) wsURL(path string, q url.Values) string {
	ref := *c.baseURL
	if ref.Scheme == "https" {
		ref.Scheme = "wss"
	} else {
		ref.Scheme = "ws"
	}
	ref.Path = path
	ref.RawQuery = q.Encode()
	return ref.String()
}

func (c *Client) ensureLogConn(ctx context.Context, level string) error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.logConn != nil {
		return nil
	}
	headers := http.Header{}
	if c.secret != "" {
		headers.Set("Authorization", "Bearer "+c.secret)
	}
	dialer := *websocket.DefaultDialer
	dialer.Proxy = nil
	conn, _, err := dialer.DialContext(
		ctx,
		c.wsURL("/logs", url.Values{"level": []string{level}}),
		headers,
	)
	if err != nil {
		return err
	}
	c.logConn = conn
	return nil
}

func (c *Client) NextLog(ctx context.Context, level string) (LogEvent, error) {
	if err := c.ensureLogConn(ctx, level); err != nil {
		return LogEvent{}, err
	}

	c.wsMu.Lock()
	conn := c.logConn
	c.wsMu.Unlock()

	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetReadDeadline(deadline)
	} else {
		_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		c.wsMu.Lock()
		if c.logConn != nil {
			_ = c.logConn.Close()
			c.logConn = nil
		}
		c.wsMu.Unlock()
		return LogEvent{}, err
	}

	var event LogEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		event = LogEvent{
			Type:    "info",
			Payload: string(msg),
		}
	}
	return event, nil
}

func (c *Client) Close() {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.logConn != nil {
		_ = c.logConn.Close()
		c.logConn = nil
	}
}
