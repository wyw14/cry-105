package verifycase

import (
	"testing"

	"orbitlink/internal/api"
	"orbitlink/internal/receiver"
	"orbitlink/internal/recorder"
)

func TestRetiredHandoverCannotCloseSuccessorRecorder(t *testing.T) {
	runtime, err := api.NewRuntime(t.TempDir())
	if err != nil { t.Fatal(err) }
	value := runtime.Passes.List()[0]
	attempt, err := runtime.Attempts.Begin(value.ID,"RF-01"); if err != nil { t.Fatal(err) }
	if err:=runtime.Passes.BeginAcquisition(value.ID,attempt.ID);err!=nil{t.Fatal(err)}
	_,err=runtime.Passes.AcceptDiversity(value.ID,[]receiver.AcquisitionResult{{PassID:value.ID,AttemptID:attempt.ID,ChainID:"RF-01",Locked:true,Synced:true,Epoch:3},{PassID:value.ID,AttemptID:attempt.ID,ChainID:"RF-02",Locked:true,Synced:true,Epoch:3}});if err!=nil{t.Fatal(err)}
	segment,found:=runtime.Recorder.ActiveForPass(value.ID);if !found{t.Fatal("recording did not start")}
	session,err:=runtime.Handovers.Begin(value.ID,segment.ID,"north","south",segment.OwnerGeneration);if err!=nil{t.Fatal(err)}
	if _,err=runtime.Handovers.Transfer(session.ID);err!=nil{t.Fatal(err)}
	if err=runtime.Handovers.Cleanup(session.ID);err!=nil{t.Fatal(err)}
	for _,current:=range runtime.Recorder.List(){if current.ID==segment.ID && current.Status!=recorder.Open{t.Fatalf("retired owner closed successor recorder: %s",current.Status)}}
}
