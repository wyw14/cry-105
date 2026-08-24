package verifycase

import (
	"testing"

	"orbitlink/internal/api"
	"orbitlink/internal/receiver"
	"orbitlink/internal/recorder"
)

func TestLateAcquisitionFailureCannotDemoteNewLock(t *testing.T) {
	runtime,err:=api.NewRuntime(t.TempDir());if err!=nil{t.Fatal(err)}
	value:=runtime.Passes.List()[0]
	segment,err:=runtime.Recorder.Start(value.ID,"north",1);if err!=nil{t.Fatal(err)}
	first,err:=runtime.Attempts.Begin(value.ID,"RF-01");if err!=nil{t.Fatal(err)}
	if err=runtime.Passes.BeginAcquisition(value.ID,first.ID);err!=nil{t.Fatal(err)}
	second,err:=runtime.Attempts.Begin(value.ID,"RF-01");if err!=nil{t.Fatal(err)}
	if err=runtime.Passes.BeginAcquisition(value.ID,second.ID);err!=nil{t.Fatal(err)}
	if err=runtime.Attempts.Deliver(receiver.AcquisitionResult{PassID:value.ID,AttemptID:second.ID,ChainID:"RF-01",Locked:true,Synced:true,Epoch:2});err!=nil{t.Fatal(err)}
	if err=runtime.Attempts.Deliver(receiver.AcquisitionResult{PassID:value.ID,AttemptID:first.ID,ChainID:"RF-01",ErrorCode:"ACQ-TIMEOUT"});err!=nil{t.Fatal(err)}
	current,_:=runtime.Passes.Get(value.ID);if current.Phase!="tracking"{t.Fatalf("late failure demoted pass to %s",current.Phase)}
	for _,item:=range runtime.Recorder.List(){if item.ID==segment.ID && item.Status!=recorder.Open{t.Fatalf("late failure stopped recorder: %s",item.Status)}}
}
