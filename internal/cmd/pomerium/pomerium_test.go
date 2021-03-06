package pomerium

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/pomerium/pomerium/config"
)

func Test_setupTracing(t *testing.T) {
	tests := []struct {
		name string
		opt  *config.Options
	}{
		{"good jaeger", &config.Options{TracingProvider: "jaeger", TracingJaegerAgentEndpoint: "localhost:0", TracingJaegerCollectorEndpoint: "localhost:0"}},
		{"dont register aything", &config.Options{}},
		{"bad provider", &config.Options{TracingProvider: "bad provider"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTracing(context.Background(), tt.opt)
		})
	}
}

func Test_setupMetrics(t *testing.T) {
	tests := []struct {
		name string
		opt  *config.Options
	}{
		{"dont register aything", &config.Options{}},
		{"good metrics server", &config.Options{MetricsAddr: "localhost:0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT)
			defer signal.Stop(c)
			setupMetrics(context.Background(), tt.opt)
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			waitSig(t, c, syscall.SIGINT)
		})
	}
}

func waitSig(t *testing.T, c <-chan os.Signal, sig os.Signal) {
	select {
	case s := <-c:
		if s != sig {
			t.Fatalf("signal was %v, want %v", s, sig)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for %v", sig)
	}
}

func Test_run(t *testing.T) {
	os.Clearenv()
	t.Parallel()
	tests := []struct {
		name           string
		configFileFlag string
		wantErr        bool
	}{
		{"nil configuration", "", true},
		{"bad proxy no authenticate url", `
		{
			"address": ":9433",
			"grpc_address": ":9444",
			"insecure_server": true,
			"authorize_service_url": "https://authorize.corp.example",
			"shared_secret": "YixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"cookie_secret": "zixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "proxy",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
		{"bad authenticate no cookie secret", `
		{
			"address": ":9433",
			"grpc_address": ":9444",
			"insecure_server": true,
			"authenticate_service_url": "https://authenticate.corp.example",
			"shared_secret": "YixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "authenticate",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
		{"bad authorize service bad shared key", `
		{
			"address": ":9433",
			"grpc_address": ":9444",
			"insecure_server": true,
			"authorize_service_url": "https://authorize.corp.example",
			"shared_secret": "^^^",
			"cookie_secret": "zixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "authorize",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
		{"bad http port", `
		{
			"address": ":-1",
			"grpc_address": ":9444",
			"grpc_insecure": true,
			"insecure_server": true,
			"authorize_service_url": "https://authorize.corp.example",
			"authenticate_service_url": "https://authenticate.corp.example",
			"shared_secret": "YixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"cookie_secret": "zixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "proxy",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
		{"bad redirect port", `
		{
			"address": ":9433",
			"http_redirect_addr":":-1",
			"grpc_address": ":9444",
			"grpc_insecure": true,
			"insecure_server": true,
			"authorize_service_url": "https://authorize.corp.example",
			"authenticate_service_url": "https://authenticate.corp.example",
			"shared_secret": "YixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"cookie_secret": "zixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "proxy",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
		{"bad metrics port ", `
		{
			"address": ":9433",
			"metrics_address": ":-1",
			"grpc_insecure": true,
			"insecure_server": true,
			"authorize_service_url": "https://authorize.corp.example",
			"authenticate_service_url": "https://authenticate.corp.example",
			"shared_secret": "YixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"cookie_secret": "zixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "proxy",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
		{"malformed tracing provider", `
		{
			"tracing_provider": "bad tracing provider",
			"address": ":9433",
			"grpc_address": ":9444",
			"grpc_insecure": true,
			"insecure_server": true,
			"authorize_service_url": "https://authorize.corp.example",
			"authenticate_service_url": "https://authenticate.corp.example",
			"shared_secret": "YixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"cookie_secret": "zixWi1MYh77NMECGGIJQevoonYtVF+ZPRkQZrrmeRqM=",
			"services": "proxy",
			"policy": [{ "from": "https://pomerium.io", "to": "https://httpbin.org" }]
		  }
		`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile(os.TempDir(), "*.json")
			if err != nil {
				t.Fatal("Cannot create temporary file", err)
			}
			defer os.Remove(tmpFile.Name())
			fn := tmpFile.Name()
			if _, err := tmpFile.Write([]byte(tt.configFileFlag)); err != nil {
				tmpFile.Close()
				t.Fatal(err)
			}
			configFile := fn

			ctx, clearTimeout := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer clearTimeout()

			err = Run(ctx, configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
