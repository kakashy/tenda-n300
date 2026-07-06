package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, loginOK bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/goform/getstok":
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
		case "/index.html":
			w.Write([]byte("ok"))
		case "/login/Auth":
			if loginOK {
				w.Header().Set("Location", "/index.html")
				w.WriteHeader(http.StatusFound)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"errCode":"1"}`))
			}
		case "/goform/getQos":
			json.NewEncoder(w).Encode(qosResponse{
				OnlineList: []qosDevice{
					{Hostname: "phone", MAC: "aa:bb:cc:dd:ee:01", IP: "192.168.1.10", ConnectType: "WiFi", Access: "true"},
					{Hostname: "laptop", MAC: "aa:bb:cc:dd:ee:02", IP: "192.168.1.11", ConnectType: "WiFi", Access: "true"},
				},
				BlackList: []qosDevice{
					{Hostname: "blocked-device", MAC: "aa:bb:cc:dd:ee:ff", IP: "0.0.0.0", ConnectType: "WiFi", Access: "false"},
				},
			})
		case "/goform/setQos":
			var res setQosResponse
			body, _ := json.Marshal(r.PostFormValue("qosList"))
			if strings.Contains(string(body), "target") {
				res.ErrCode = "0"
			} else {
				// accept anything with at least one entry
				if r.PostFormValue("qosList") != "" {
					res.ErrCode = "0"
				} else {
					res.ErrCode = "1"
				}
			}
			json.NewEncoder(w).Encode(res)
		case "/goform/sysReboot":
			w.WriteHeader(http.StatusOK)
		case "/goform/sysRestore":
			w.WriteHeader(http.StatusOK)
		case "/cgi-bin/DownloadCfg/RouterCfm.cfg":
			w.Write([]byte("mock-config-data"))
		case "/cgi-bin/DownloadSyslog/RouterSystem.log":
			w.Write([]byte("mock-syslog-data"))
		case "/login.html":
			w.Write([]byte(`<html><script>var model_name="N300";var hardware_version="V1.0";var firmware_version="V5.07.68";</script></html>`))
		case "/goform/getInternetCfg":
			json.NewEncoder(w).Encode(map[string]string{
				"wanType":    "dhcp",
				"dnsServer":  "8.8.8.8",
				"dnsServer1": "8.8.4.4",
				"gateway":    "10.0.0.1",
			})
		case "/cgi-bin/UploadCfg":
			w.Write([]byte(`{"errCode":0}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestNewRouterClient(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewRouterClientLoginFails(t *testing.T) {
	srv := newTestServer(t, false)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	_, err := NewRouterClient(ip, "wrong")
	if err == nil {
		t.Fatal("expected login to fail")
	}
}

func TestGetDevices(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	devices, err := client.GetDevices()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}

	expected := map[string]bool{
		"aa:bb:cc:dd:ee:01": true,
		"aa:bb:cc:dd:ee:02": true,
		"aa:bb:cc:dd:ee:ff": false,
	}
	for _, d := range devices {
		access, ok := expected[d.MAC]
		if !ok {
			t.Errorf("unexpected device MAC: %s", d.MAC)
			continue
		}
		if d.Access != access {
			t.Errorf("device %s: expected access=%v, got %v", d.MAC, access, d.Access)
		}
	}
}

func TestBlockMAC(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.BlockMAC("aa:bb:cc:dd:ee:01"); err != nil {
		t.Fatal(err)
	}
}

func TestUnblockMAC(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UnblockMAC("aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
}

func TestReboot(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Reboot(); err != nil {
		t.Fatal(err)
	}
}

func TestReset(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Reset(); err != nil {
		t.Fatal(err)
	}
}

func TestBackupConfig(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.BackupConfig()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mock-config-data" {
		t.Fatalf("expected mock-config-data, got %s", string(data))
	}
}

func TestExportSyslog(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.ExportSyslog()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mock-syslog-data" {
		t.Fatalf("expected mock-syslog-data, got %s", string(data))
	}
}

func TestGetDevicesJSONParse(t *testing.T) {
	// Test the JSON structure directly to catch field name drift
	raw := `{
		"onlineList": [
			{"qosListHostname": "pc", "qosListMac": "11:22:33:44:55:66", "qosListIP": "10.0.0.2", "qosListConnectType": "Ethernet", "qosListUpLimit": "", "qosListDownLimit": "", "qosListAccess": "true", "qosListRemark": ""}
		],
		"blackList": []
	}`
	var qos qosResponse
	if err := json.Unmarshal([]byte(raw), &qos); err != nil {
		t.Fatal(err)
	}
	if len(qos.OnlineList) != 1 {
		t.Fatalf("expected 1 online device, got %d", len(qos.OnlineList))
	}
	if qos.OnlineList[0].MAC != "11:22:33:44:55:66" {
		t.Fatalf("expected MAC 11:22:33:44:55:66, got %s", qos.OnlineList[0].MAC)
	}
}

func TestSetQosResponse(t *testing.T) {
	tests := []struct {
		json string
		want string
	}{
		{`{"errCode":"0"}`, "0"},
		{`{"errCode":"1"}`, "1"},
	}
	for _, tc := range tests {
		var res setQosResponse
		if err := json.Unmarshal([]byte(tc.json), &res); err != nil {
			t.Fatal(err)
		}
		if res.ErrCode != tc.want {
			t.Fatalf("expected errCode %s, got %s", tc.want, res.ErrCode)
		}
	}
}

func TestRestoreConfig(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.RestoreConfig("router_test.go"); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreConfigSpacedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/goform/getstok" {
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
			return
		}
		if r.URL.Path == "/index.html" {
			w.Write([]byte("ok"))
			return
		}
		if r.URL.Path == "/login/Auth" {
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/cgi-bin/UploadCfg" {
			w.Write([]byte(`{"errCode": 0}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.RestoreConfig("router_test.go"); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreConfigFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/goform/getstok" {
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
			return
		}
		if r.URL.Path == "/index.html" {
			w.Write([]byte("ok"))
			return
		}
		if r.URL.Path == "/login/Auth" {
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/cgi-bin/UploadCfg" {
			w.Write([]byte(`{"errCode": 1}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.RestoreConfig("router_test.go"); err == nil {
		t.Fatal("expected error for errCode 1")
	}
}

func TestGetFirmwareInfo(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	info := client.GetFirmwareInfo()
	if info == nil {
		t.Fatal("expected non-nil FirmwareInfo")
	}
	if info.Model != "N300" {
		t.Errorf("expected Model=N300, got %s", info.Model)
	}
	if info.Version != "V5.07.68" {
		t.Errorf("expected Version=V5.07.68, got %s", info.Version)
	}
	if info.Hardware != "V1.0" {
		t.Errorf("expected Hardware=V1.0, got %s", info.Hardware)
	}
	if info.DefaultDNS != "8.8.8.8" {
		t.Errorf("expected DefaultDNS=8.8.8.8, got %s", info.DefaultDNS)
	}
	if info.AltDNS != "8.8.4.4" {
		t.Errorf("expected AltDNS=8.8.4.4, got %s", info.AltDNS)
	}
	if info.ConnectionType != "DHCP" {
		t.Errorf("expected ConnectionType=DHCP, got %s", info.ConnectionType)
	}
	if info.Gateway != "10.0.0.1" {
		t.Errorf("expected Gateway=10.0.0.1, got %s", info.Gateway)
	}
}

func TestExtractJSVar(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"simple", `<script>var model_name="N300";</script>`, "N300"},
		{"with spaces", `<script>var model_name = "N300";</script>`, "N300"},
		{"with single quotes", `<script>var model_name='N300';</script>`, "N300"},
		{"missing var", `<script>var other="value";</script>`, ""},
		{"malformed", `<script>var model_name=N300</script>`, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractJSVar([]byte(tc.html), "model_name")
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestInternetCfgResponse(t *testing.T) {
	tests := []struct {
		json string
		want internetCfgResponse
	}{
		{
			`{"wanType":"dhcp","dnsServer":"8.8.8.8","dnsServer1":"8.8.4.4","gateway":"10.0.0.1"}`,
			internetCfgResponse{
				WanType:    "dhcp",
				DNSServer:  "8.8.8.8",
				DNSServer1: "8.8.4.4",
				Gateway:    "10.0.0.1",
			},
		},
		{
			`{"wanType":"pppoe","dnsServer":"1.1.1.1","dnsServer1":"","gateway":"0.0.0.0"}`,
			internetCfgResponse{
				WanType:    "pppoe",
				DNSServer:  "1.1.1.1",
				DNSServer1: "",
				Gateway:    "0.0.0.0",
			},
		},
	}
	for _, tc := range tests {
		var res internetCfgResponse
		if err := json.Unmarshal([]byte(tc.json), &res); err != nil {
			t.Fatal(err)
		}
		if res.WanType != tc.want.WanType {
			t.Errorf("expected WanType=%s, got %s", tc.want.WanType, res.WanType)
		}
		if res.DNSServer != tc.want.DNSServer {
			t.Errorf("expected DNSServer=%s, got %s", tc.want.DNSServer, res.DNSServer)
		}
		if res.DNSServer1 != tc.want.DNSServer1 {
			t.Errorf("expected DNSServer1=%s, got %s", tc.want.DNSServer1, res.DNSServer1)
		}
		if res.Gateway != tc.want.Gateway {
			t.Errorf("expected Gateway=%s, got %s", tc.want.Gateway, res.Gateway)
		}
	}
}

func TestGetFirmwareInfoPartialData(t *testing.T) {
	// Only login page works, internetCfg fails
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/goform/getstok" {
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
			return
		}
		if r.URL.Path == "/index.html" {
			w.Write([]byte("ok"))
			return
		}
		if r.URL.Path == "/login/Auth" {
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/login.html" {
			w.Write([]byte(`<script>var model_name="N300";var hardware_version="V1.0";var firmware_version="V5.07.68";</script>`))
			return
		}
		if r.URL.Path == "/goform/getInternetCfg" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	info := client.GetFirmwareInfo()
	if info.Model != "N300" {
		t.Errorf("expected Model=N300, got %s", info.Model)
	}
	if info.DefaultDNS != "" {
		t.Errorf("expected empty DefaultDNS when endpoint fails, got %s", info.DefaultDNS)
	}
}

func TestGetFirmwareInfoAllFail(t *testing.T) {
	// Both login page and internetCfg fail
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/goform/getstok" {
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
			return
		}
		if r.URL.Path == "/index.html" {
			w.Write([]byte("ok"))
			return
		}
		if r.URL.Path == "/login/Auth" {
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
			return
		}
		// Both /login.html and /goform/getInternetCfg return 404
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	info := client.GetFirmwareInfo()
	if info == nil {
		t.Fatal("expected non-nil FirmwareInfo even when all sources fail")
	}
	if info.Model != "" {
		t.Errorf("expected empty Model, got %s", info.Model)
	}
}

func TestDefaultGatewayIP(t *testing.T) {
	ip, err := defaultGatewayIP()
	if err != nil {
		t.Skipf("no default gateway: %v", err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("invalid IP from defaultGatewayIP: %s", ip)
	}
}
