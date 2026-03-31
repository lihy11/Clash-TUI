package mihomo

import "encoding/json"

type VersionResponse struct {
	Version string `json:"version"`
}

type ConfigResponse struct {
	Mode           string    `json:"mode"`
	LogLevel       string    `json:"log-level"`
	IPv6           bool      `json:"ipv6"`
	SystemProxy    bool      `json:"system-proxy"`
	Tun            TunConfig `json:"tun"`
	HasSystemProxy bool      `json:"-"`
}

func (c *ConfigResponse) UnmarshalJSON(data []byte) error {
	type alias ConfigResponse
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*c = ConfigResponse(tmp)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		_, c.HasSystemProxy = raw["system-proxy"]
	}
	return nil
}

type UpdateConfigRequest struct {
	Mode        string     `json:"mode,omitempty"`
	SystemProxy *bool      `json:"system-proxy,omitempty"`
	Tun         *TunConfig `json:"tun,omitempty"`
}

type TunConfig struct {
	Enable bool `json:"enable"`
}

func (t *TunConfig) UnmarshalJSON(data []byte) error {
	// /configs may return tun as a bool or an object depending on kernel/runtime.
	var v bool
	if err := json.Unmarshal(data, &v); err == nil {
		t.Enable = v
		return nil
	}
	var obj struct {
		Enable bool `json:"enable"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		t.Enable = obj.Enable
		return nil
	}
	t.Enable = false
	return nil
}

type ProxiesResponse struct {
	Proxies map[string]Proxy `json:"proxies"`
}

type Proxy struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now"`
	All     []string       `json:"all"`
	History []DelayHistory `json:"history"`
}

type DelayHistory struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

type SelectProxyRequest struct {
	Name string `json:"name"`
}

type DelayResponse struct {
	Delay int `json:"delay"`
}

type RulesResponse struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

type ConnectionsResponse struct {
	Connections []Connection `json:"connections"`
}

type Connection struct {
	ID          string             `json:"id"`
	Upload      int64              `json:"upload"`
	Download    int64              `json:"download"`
	Rule        string             `json:"rule"`
	RulePayload string             `json:"rulePayload"`
	Chains      []string           `json:"chains"`
	Metadata    ConnectionMetadata `json:"metadata"`
}

type ConnectionMetadata struct {
	Host            string `json:"host"`
	DestinationIP   string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Network         string `json:"network"`
	Type            string `json:"type"`
	Process         string `json:"process"`
}

type ProvidersResponse struct {
	Providers map[string]Provider `json:"providers"`
}

type Provider struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	VehicleType string   `json:"vehicleType"`
	UpdatedAt   string   `json:"updatedAt"`
	Proxies     ProxySet `json:"proxies"`
}

type ProviderProxy struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Alive   bool           `json:"alive"`
	History []DelayHistory `json:"history"`
}

type ProxySet []ProviderProxy

func (p *ProxySet) UnmarshalJSON(data []byte) error {
	var objs []ProviderProxy
	if err := json.Unmarshal(data, &objs); err == nil {
		*p = objs
		return nil
	}

	var names []string
	if err := json.Unmarshal(data, &names); err == nil {
		out := make([]ProviderProxy, 0, len(names))
		for _, n := range names {
			out = append(out, ProviderProxy{Name: n})
		}
		*p = out
		return nil
	}

	*p = nil
	return nil
}

type LogEvent struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}
