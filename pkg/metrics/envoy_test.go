package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEnvoyObserver_GetRequestSuccessRate(t *testing.T) {
	expected := ` sum( rate( envoy_cluster_upstream_rq{ kubernetes_namespace="default", kubernetes_pod_name=~"podinfo-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)", envoy_response_code!~"5.*" }[1m] ) ) / sum( rate( envoy_cluster_upstream_rq{ kubernetes_namespace="default", kubernetes_pod_name=~"podinfo-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)" }[1m] ) ) * 100`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promql := r.URL.Query()["query"][0]
		if promql != expected {
			t.Errorf("\nGot %s \nWanted %s", promql, expected)
		}

		json := `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1,"100"]}]}}`
		w.Write([]byte(json))
	}))
	defer ts.Close()

	client, err := NewPrometheusClient(ts.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	observer := &EnvoyObserver{
		client: client,
	}

	val, err := observer.GetRequestSuccessRate("podinfo", "default", "1m")
	if err != nil {
		t.Fatal(err.Error())
	}

	if val != 100 {
		t.Errorf("Got %v wanted %v", val, 100)
	}
}

func TestEnvoyObserver_GetRequestDuration(t *testing.T) {
	expected := ` histogram_quantile( 0.99, sum( rate( envoy_cluster_upstream_rq_time_bucket{ kubernetes_namespace="default", kubernetes_pod_name=~"podinfo-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)" }[1m] ) ) by (le) )`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promql := r.URL.Query()["query"][0]
		if promql != expected {
			t.Errorf("\nGot %s \nWanted %s", promql, expected)
		}

		json := `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1,"100"]}]}}`
		w.Write([]byte(json))
	}))
	defer ts.Close()

	client, err := NewPrometheusClient(ts.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	observer := &EnvoyObserver{
		client: client,
	}

	val, err := observer.GetRequestDuration("podinfo", "default", "1m")
	if err != nil {
		t.Fatal(err.Error())
	}

	if val != 100*time.Millisecond {
		t.Errorf("Got %v wanted %v", val, 100*time.Millisecond)
	}
}
