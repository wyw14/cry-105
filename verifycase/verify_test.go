package verifycase

import (
	"sync/atomic"
	"testing"
	"time"

	"orbitlink/internal/api"
	passsvc "orbitlink/internal/pass"
)

func TestPassWindowPublishesOneEphemerisRevision(t *testing.T) {
	runtime, err := api.NewRuntime(t.TempDir())
	if err != nil { t.Fatal(err) }
	value := runtime.Passes.List()[0]
	updated := passsvc.Window{AOS:value.Window.LOS.Add(time.Minute),LOS:value.Window.LOS.Add(4*time.Minute),Revision:value.Window.Revision+1}
	var mixed atomic.Bool
	done := make(chan error,1)
	go func(){done<-runtime.WindowRecompute.Apply(value.ID,updated)}()
	deadline:=time.Now().Add(250*time.Millisecond)
	for time.Now().Before(deadline){
		window,found:=runtime.Windows.Get(value.ID)
		if found && !window.LOS.After(window.AOS){mixed.Store(true);break}
		select{case err:=<-done:if err!=nil{t.Fatal(err)};if mixed.Load(){t.Fatal("scheduler-visible pass window mixed ephemeris revisions")};return;default:}
	}
	if err:=<-done;err!=nil{t.Fatal(err)}
	if mixed.Load(){t.Fatal("scheduler-visible pass window mixed ephemeris revisions")}
}
