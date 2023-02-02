package poc

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tbeets/npoci"
	"github.com/tbeets/poci"
)

func TestStartRetailFleet(t *testing.T) {
	wd, _ := os.Getwd()
	rf := StartRetailFleet(wd + "/..")
	poci.Require_True(t, len(rf.servers) == 1)
	defer StopRetailFleet(rf)
}

var testAStr = `
{
  "name": "testA",
  "subjects": [
    "foo.*"
  ],
  "retention": "limits",
  "max_consumers": -1,
  "max_msgs_per_subject": -1,
  "max_msgs": -1,
  "max_bytes": -1,
  "max_age": 0,
  "max_msg_size": -1,
  "storage": "file",
  "discard": "old",
  "num_replicas": 1,
  "duplicate_window": 120000000000,
  "sealed": false,
  "deny_delete": false,
  "deny_purge": false,
  "allow_rollup_hdrs": false,
  "allow_direct": false,
  "mirror_direct": false
}
`

var testBStr = `
{
  "name": "testB",
  "sources": [
    {
      "name": "testA",
      "filter_subject": "foo.b",
      "external": {
         "api": "$JS.testA.API",
         "deliver": "testB"
      }
    }
  ],
  "retention": "limits",
  "max_consumers": -1,
  "max_msgs_per_subject": -1,
  "max_msgs": -1,
  "max_bytes": -1,
  "max_age": 0,
  "max_msg_size": -1,
  "storage": "file",
  "discard": "old",
  "num_replicas": 1,
  "duplicate_window": 120000000000,
  "sealed": false,
  "deny_delete": false,
  "deny_purge": false,
  "allow_rollup_hdrs": false,
  "allow_direct": false,
  "mirror_direct": false
}
`

func TestCrossAccountSourcing(t *testing.T) {
	wd, _ := os.Getwd()

	deleteTempState()

	rf := StartRetailFleet(wd + "/..")
	poci.Require_True(t, len(rf.servers) == 1)
	defer StopRetailFleet(rf)
	s := rf.servers[0]

	cA, jscA := npoci.JsClientConnect(t, s, nats.UserInfo("user-testA", "s3cr3t"))
	poci.Require_True(t, cA != nil && jscA != nil)
	defer cA.Close()

	var err error
	var cfgA nats.StreamConfig
	var cfgB nats.StreamConfig

	err = json.Unmarshal([]byte(testAStr), &cfgA)
	if err != nil {
		t.Fatalf("error unmarshalling stream testA: %s", err.Error())
	}

	_, err = jscA.AddStream(&cfgA)
	if err != nil {
		t.Fatalf("error adding stream testA: %s", err.Error())
	}

	cB, jscB := npoci.JsClientConnect(t, s, nats.UserInfo("user-testB", "s3cr3t"))
	poci.Require_True(t, cB != nil && jscB != nil)
	defer cB.Close()

	err = json.Unmarshal([]byte(testBStr), &cfgB)
	if err != nil {
		t.Fatalf("error unmarshalling stream testB: %s", err.Error())
	}

	_, err = jscB.AddStream(&cfgB)
	if err != nil {
		t.Fatalf("error adding stream testB: %s", err.Error())
	}

	// Downstream should not get foo.a
	_, err = jscA.Publish("foo.a", []byte("you should not see"))
	if err != nil {
		t.Fatalf("expected to be able to publish foo.a in testA: %s", err.Error())
	}

	// Downstream should get foo.b
	_, err = jscA.Publish("foo.b", []byte("be my guest"))
	if err != nil {
		t.Fatalf("expected to be able to publish foo.b in testA: %s", err.Error())
	}

	time.Sleep(100 * time.Millisecond)
	poci.CheckFor(t, 2*time.Second, 100*time.Millisecond, func() error {
		infoB, err := jscB.StreamInfo("testB")
		poci.Require_NoError(t, err)

		m := infoB.State.Msgs
		if m == 1 {
			return nil
		}
		return fmt.Errorf("expected 1 message in downstream, got %d", m)
	})

	rmsg, err := jscB.GetLastMsg("testB", "foo.*")
	poci.Require_NoError(t, err)
	poci.Require_False(t, rmsg == nil)
	poci.Require_False(t, rmsg.Subject == "foo.a")
}

func deleteTempState() {
	_ = poci.RemoveContents("/tmp/jetstream")
}
