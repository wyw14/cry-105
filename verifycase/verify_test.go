package verifycase

import (
	"testing"

	"orbitlink/internal/audit"
	"orbitlink/internal/decoder"
	"orbitlink/internal/receiver"
	"orbitlink/internal/recorder"
	"orbitlink/internal/store"
)

func TestReacquisitionStartsFreshDecoderEpoch(t *testing.T) {
	paths:=store.NewPaths(t.TempDir());trail:=audit.NewTrail(store.NewAppendLog(paths))
	manager,err:=recorder.NewManager(paths,trail);if err!=nil{t.Fatal(err)}
	segment,err:=manager.Start("pass-1","north",1);if err!=nil{t.Fatal(err)}
	decoderSession,err:=decoder.NewSession("pass-1",segment.ID,8,manager,trail);if err!=nil{t.Fatal(err)}
	session:=receiver.NewSession("pass-1",decoderSession)
	firstEpoch,err:=session.OnLock("attempt-1");if err!=nil{t.Fatal(err)}
	if frames,err:=session.Feed("attempt-1",[]byte{1,2,3,4});err!=nil||frames!=0{t.Fatalf("first half: %d %v",frames,err)}
	if err=session.OnCarrierLoss("attempt-1");err!=nil{t.Fatal(err)}
	secondEpoch,err:=session.OnLock("attempt-2");if err!=nil{t.Fatal(err)}
	frames,err:=session.Feed("attempt-2",[]byte{5,6,7,8});if err!=nil{t.Fatal(err)}
	if secondEpoch<=firstEpoch{t.Fatalf("decoder epoch did not advance: %d -> %d",firstEpoch,secondEpoch)}
	if frames!=0{t.Fatalf("new attempt completed a frame from the prior lock: %d",frames)}
	current,_:=manager.ActiveForPass("pass-1");if current.FramesBuffered!=0{t.Fatalf("corrupt cross-epoch frame reached recorder: %d",current.FramesBuffered)}
}
