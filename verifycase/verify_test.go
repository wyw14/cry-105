package verifycase

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"orbitlink/internal/api"
	passsvc "orbitlink/internal/pass"
)

func TestPassExtensionAndAcquisitionNeverOverlapAntenna(t *testing.T) {
	runtime, err := api.NewRuntime(t.TempDir())
	if err != nil { t.Fatal(err) }
	active := runtime.Passes.List()[0]
	attempt, err := runtime.Attempts.Begin(active.ID, "RF-01")
	if err != nil { t.Fatal(err) }
	if err := runtime.Passes.BeginAcquisition(active.ID, attempt.ID); err != nil { t.Fatal(err) }
	next, err := runtime.Passes.Create(passsvc.CreateRequest{
		SatelliteID: "SAT-C07", AntennaID: active.AntennaID, RFChains: []string{"RF-03"},
		RequiredReceiverChains: []string{"RF-03"}, AOS: active.Window.LOS.Add(time.Minute),
		LOS: active.Window.LOS.Add(3*time.Minute), Revision: 1, OwnerStation: "north",
	})
	if err != nil { t.Fatal(err) }
	if err := runtime.WindowRecompute.Apply(next.ID, next.Window); err != nil { t.Fatal(err) }
	router, err := api.NewRouter(runtime)
	if err != nil { t.Fatal(err) }
	server := httptest.NewServer(router)
	defer server.Close()
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var group sync.WaitGroup
	post := func(target, body string) {
		defer group.Done()
		<-start
		response, requestErr := http.Post(server.URL+target, "application/json", bytes.NewBufferString(body))
		if requestErr != nil { statuses <- 0; return }
		defer response.Body.Close()
		statuses <- response.StatusCode
	}
	group.Add(2)
	go post("/api/passes/"+active.ID+"/extend", `{"seconds":180}`)
	go post("/api/passes/"+next.ID+"/reserve", `{}`)
	close(start)
	group.Wait()
	close(statuses)
	successes := 0
	for status := range statuses {
		if status >= 200 && status < 300 { successes++ }
	}
	if successes != 1 {
		t.Fatalf("expected one timeline mutation to commit, got %d successful operations", successes)
	}
}
