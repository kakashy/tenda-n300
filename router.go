package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Device struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Type     string `json:"type"`
	Access   bool   `json:"access"`
}

type qosResponse struct {
	OnlineList []qosDevice `json:"onlineList"`
	BlackList  []qosDevice `json:"blackList"`
}

type qosDevice struct {
	Hostname    string `json:"qosListHostname"`
	Remark      string `json:"qosListRemark"`
	MAC         string `json:"qosListMac"`
	IP          string `json:"qosListIP"`
	ConnectType string `json:"qosListConnectType"`
	UpLimit     string `json:"qosListUpLimit"`
	DownLimit   string `json:"qosListDownLimit"`
	Access      string `json:"qosListAccess"`
}

type stokResponse struct {
	Random string `json:"random"`
}

type FirmwareInfo struct {
	Version        string `json:"version"`
	Uptime         string `json:"uptime"`
	DefaultDNS     string `json:"defaultdns"`
	AltDNS         string `json:"altdns"`
	ConnectionType string `json:"connectiontype"`
	Gateway        string `json:"gateway"`
	WanIP          string `json:"wanip"`
	WanMAC         string `json:"wanmac"`
}

type statusModulesResponse struct {
	SystemInfo *systemInfoModule `json:"systemInfo"`
}

type systemInfoModule struct {
	WanType         string `json:"wanType"`
	WanConnectTime  string `json:"wanConnectTime"`
	LanIP           string `json:"lanIP"`
	MacHost         string `json:"macHost"`
	SoftVersion     string `json:"softVersion"`
	StatusWanDns1   string `json:"statusWanDns1"`
	StatusWanDns2   string `json:"statusWanDns2"`
	StatusWanGaterway string `json:"statusWanGaterway"`
	StatusWanIP     string `json:"statusWanIP"`
	StatusWanMAC    string `json:"statusWanMAC"`
	StatusWanMask   string `json:"statusWanMask"`
}

type setQosResponse struct {
	ErrCode string `json:"errCode"`
}

type WiFiSettings struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Channel    string `json:"channel"`
	Encryption string `json:"encrypt"`
	Band       string `json:"band,omitempty"`
	WPS        string `json:"wps,omitempty"`
	Broadcast  string `json:"broadcast,omitempty"`
}

type wifiResponse struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Channel    string `json:"channel"`
	Encryption string `json:"encrypt"`
	Band       string `json:"band"`
	WPS        string `json:"wps"`
	Broadcast  string `json:"broadcast"`
}

type setWifiResponse struct {
	ErrCode string `json:"errCode"`
}

const clientTimeout = 10 * time.Second
const pingTimeout = 5 * time.Second

// wrapTimeoutError returns a user-friendly message if err is a network timeout.
func wrapTimeoutError(err error) error {
	if err == nil {
		return nil
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fmt.Errorf("connection to router timed out (is the router online?)")
	}
	return err
}

type RouterClient struct {
	baseURL string
	client  *http.Client
}

func NewRouterClient(ip, password string) (*RouterClient, error) {
	baseURL := fmt.Sprintf("http://%s", ip)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &RouterClient{
		baseURL: baseURL,
		client:  &http.Client{Jar: jar, Timeout: clientTimeout},
	}
	if err := c.login(password); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RouterClient) getStok() (string, error) {
	resp, err := c.client.Get(c.baseURL + "/goform/getstok")
	if err != nil {
		return "", wrapTimeoutError(fmt.Errorf("getstok: %w", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var s stokResponse
	if err := json.Unmarshal(body, &s); err != nil {
		return "", fmt.Errorf("decode stok: %w", err)
	}
	return s.Random, nil
}

func (c *RouterClient) login(password string) error {
	for i := 0; i < 3; i++ {
		random, err := c.getStok()
		if err != nil {
			if i == 2 {
				return err
			}
			continue
		}
		b64pass := base64.StdEncoding.EncodeToString([]byte(password))
		hash := md5.Sum([]byte(b64pass + random))
		encoded := hex.EncodeToString(hash[:])
		data := url.Values{"password": {encoded}}
		req, err := http.NewRequest("POST", c.baseURL+"/login/Auth", strings.NewReader(data.Encode()))
		if err != nil {
			if i == 2 {
				return err
			}
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "tenda-n300/1.0")
		req.Header.Set("Referer", c.baseURL+"/login.html")
		resp, err := c.client.Do(req)
		if err != nil {
			if i == 2 {
				return wrapTimeoutError(fmt.Errorf("login: %w", err))
			}
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if strings.Contains(resp.Request.URL.String(), "index.html") {
			return nil
		}
	}
	return fmt.Errorf("login failed (wrong password or rate-limited)")
}

func (c *RouterClient) get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", c.baseURL+"/index.html")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wrapTimeoutError(err)
	}
	return resp, nil
}

func (c *RouterClient) post(path string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.baseURL+path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", c.baseURL+"/index.html")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wrapTimeoutError(err)
	}
	return resp, nil
}

func (c *RouterClient) GetDevices() ([]Device, error) {
	resp, err := c.get("/goform/getQos?modules=onlineList,blackList")
	if err != nil {
		return nil, fmt.Errorf("getQos: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var qos qosResponse
	if err := json.Unmarshal(body, &qos); err != nil {
		return nil, fmt.Errorf("decode QoS: %w", err)
	}
	devices := make([]Device, 0, len(qos.OnlineList)+len(qos.BlackList))
	for _, d := range qos.OnlineList {
		devices = append(devices, Device{
			Hostname: d.Hostname,
			IP:       d.IP,
			MAC:      d.MAC,
			Type:     d.ConnectType,
			Access:   true,
		})
	}
	for _, d := range qos.BlackList {
		devices = append(devices, Device{
			Hostname: d.Hostname,
			IP:       d.IP,
			MAC:      d.MAC,
			Type:     d.ConnectType,
			Access:   false,
		})
	}
	return devices, nil
}

func (c *RouterClient) BlockMAC(mac string) error {
	return c.setMACAccess(mac, "block")
}

func (c *RouterClient) UnblockMAC(mac string) error {
	return c.setMACAccess(mac, "unblock")
}

func (c *RouterClient) setMACAccess(targetMAC, action string) error {
	// Validate and normalize MAC address
	if err := ValidateMAC(targetMAC); err != nil {
		return err
	}
	targetMAC = NormalizeMAC(targetMAC)

	resp, err := c.get("/goform/getQos?modules=onlineList,blackList")
	if err != nil {
		return fmt.Errorf("fetch QoS: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	var qos qosResponse
	if err := json.Unmarshal(body, &qos); err != nil {
		return fmt.Errorf("decode QoS: %w", err)
	}

	var b strings.Builder

	for _, d := range qos.OnlineList {
		mac := NormalizeMAC(d.MAC)
		access := d.Access
		if mac == targetMAC {
			if action == "block" {
				access = "false"
			} else {
				access = "true"
			}
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n", d.Hostname, d.Remark, d.MAC, d.UpLimit, d.DownLimit, access))
	}
	for _, d := range qos.BlackList {
		mac := NormalizeMAC(d.MAC)
		if mac == targetMAC && action == "unblock" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\tfalse\n", d.Hostname, d.Remark, d.MAC, d.UpLimit, d.DownLimit))
	}

	qosList := strings.TrimRight(b.String(), "\n")
	data := url.Values{
		"module1": {"qosList"},
		"qosList": {qosList},
	}
	resp2, err := c.post("/goform/setQos", data)
	if err != nil {
		return fmt.Errorf("setQos: %w", err)
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return err
	}
	var res setQosResponse
	if err := json.Unmarshal(body2, &res); err != nil {
		return fmt.Errorf("decode setQos: %w", err)
	}
	if res.ErrCode != "0" {
		return fmt.Errorf("QoS rejected, errCode: %s", res.ErrCode)
	}
	return nil
}

func (c *RouterClient) Reboot() error {
	data := url.Values{"module1": {"sysOperate"}, "action": {"reboot"}}
	resp, err := c.post("/goform/sysReboot", data)
	if err != nil {
		return fmt.Errorf("reboot: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *RouterClient) Reset() error {
	data := url.Values{"module1": {"sysOperate"}, "action": {"restore"}}
	resp, err := c.post("/goform/sysRestore", data)
	if err != nil {
		return fmt.Errorf("reset: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *RouterClient) BackupConfig() ([]byte, error) {
	resp, err := c.get("/cgi-bin/DownloadCfg/RouterCfm.cfg")
	if err != nil {
		return nil, fmt.Errorf("backup: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *RouterClient) RestoreConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("inportFile", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", c.baseURL+"/cgi-bin/UploadCfg", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", c.baseURL+"/index.html")
	resp, err := c.client.Do(req)
	if err != nil {
		return wrapTimeoutError(fmt.Errorf("restore: %w", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("restore: read response: %w", err)
	}

	var result struct {
		ErrCode json.RawMessage `json:"errCode"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("restore: parse response: %w", err)
	}

	code := strings.Trim(string(result.ErrCode), `" `)
	if code == "0" || code == "100" {
		return nil
	}
	return fmt.Errorf("restore failed: server returned unexpected response")
}

func (c *RouterClient) GetFirmwareInfo() (*FirmwareInfo, error) {
	status, err := c.getStatus()
	if err != nil {
		return nil, err
	}

	info := &FirmwareInfo{}
	if s := status.SystemInfo; s != nil {
		info.ConnectionType = translateWanType(s.WanType)
		info.Version = s.SoftVersion
		info.DefaultDNS = s.StatusWanDns1
		info.AltDNS = s.StatusWanDns2
		info.Gateway = s.StatusWanGaterway
		info.WanIP = s.StatusWanIP
		info.WanMAC = s.StatusWanMAC
		info.Uptime = formatUptime(s.WanConnectTime)
	}

	return info, nil
}

func (c *RouterClient) getStatus() (*statusModulesResponse, error) {
	path := "/goform/getStatus?modules=systemInfo"
	resp, err := c.get(path)
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read status: %w", err)
	}
	var s statusModulesResponse
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("decode status: %w", err)
	}
	return &s, nil
}

func translateWanType(t string) string {
	switch t {
	case "dhcp":
		return "Dynamic IP"
	case "pppoe":
		return "PPPoE"
	default:
		return "Static IP"
	}
}

func formatUptime(secs string) string {
	var total int
	if _, err := fmt.Sscanf(secs, "%d", &total); err != nil {
		return secs
	}
	if total <= 0 {
		return ""
	}
	d := total / 86400
	total %= 86400
	h := total / 3600
	total %= 3600
	m := total / 60
	s := total % 60

	var parts strings.Builder
	if d > 0 {
		fmt.Fprintf(&parts, "%dd ", d)
	}
	if h > 0 {
		fmt.Fprintf(&parts, "%dh ", h)
	}
	if m > 0 {
		fmt.Fprintf(&parts, "%dm ", m)
	}
	fmt.Fprintf(&parts, "%ds", s)
	return parts.String()
}

func (c *RouterClient) ExportSyslog() ([]byte, error) {
	resp, err := c.get("/cgi-bin/DownloadSyslog/RouterSystem.log")
	if err != nil {
		return nil, fmt.Errorf("syslog: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *RouterClient) GetWiFiSettings() (*WiFiSettings, error) {
	resp, err := c.get("/goform/getWifi")
	if err != nil {
		return nil, fmt.Errorf("getWifi: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read getWifi: %w", err)
	}
	var w wifiResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, fmt.Errorf("decode getWifi: %w", err)
	}
	return &WiFiSettings{
		SSID:       w.SSID,
		Password:   w.Password,
		Channel:    w.Channel,
		Encryption: w.Encryption,
		Band:       w.Band,
		WPS:        w.WPS,
		Broadcast:  w.Broadcast,
	}, nil
}

func (c *RouterClient) SetWiFiSettings(s *WiFiSettings) error {
	if s == nil {
		return fmt.Errorf("setWifi: settings cannot be nil")
	}
	if s.SSID == "" || s.Password == "" || s.Channel == "" || s.Encryption == "" {
		return fmt.Errorf("setWifi: SSID, password, channel, and encryption are required")
	}
	// Note: "SSID" is uppercase to match the Tenda goform API field name.
	// All other fields (password, channel, encrypt) are lowercase as the router expects.
	data := url.Values{
		"SSID":     {s.SSID},
		"password": {s.Password},
		"channel":  {s.Channel},
		"encrypt":  {s.Encryption},
	}
	resp, err := c.post("/goform/setWifi", data)
	if err != nil {
		return fmt.Errorf("setWifi: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read setWifi: %w", err)
	}
	var res setWifiResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("decode setWifi: %w", err)
	}
	if res.ErrCode != "0" {
		return fmt.Errorf("setWifi rejected, errCode: %s", res.ErrCode)
	}
	return nil
}

type PingResult struct {
	Reachable bool          `json:"reachable"`
	Latency   time.Duration `json:"latency"`
	APIAccess bool          `json:"api_access"`
	RouterIP  string        `json:"router_ip"`
	Error     string        `json:"error,omitempty"`
}

func PingRouter(ip string) *PingResult {
	result := &PingResult{RouterIP: ip}
	baseURL := fmt.Sprintf("http://%s", ip)

	client := &http.Client{Timeout: pingTimeout}
	start := time.Now()
	resp, err := client.Get(baseURL + "/login.html")

	if err != nil {
		result.Error = wrapTimeoutError(err).Error()
		return result
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	result.Latency = time.Since(start)
	result.Reachable = true

	req, _ := http.NewRequest("GET", baseURL+"/goform/getstok", nil)
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", baseURL+"/index.html")
	resp2, err := client.Do(req)
	if err == nil {
		io.Copy(io.Discard, resp2.Body)
		resp2.Body.Close()
		result.APIAccess = resp2.StatusCode == http.StatusOK
	}

	return result
}
