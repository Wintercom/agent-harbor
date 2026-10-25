package gate

import (
	"bytes"
	"testing"
)

type tailBlockingReader struct {
	payload []byte
	done    bool
}

func (r *tailBlockingReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, r.payload), nil
	}
	select {}
}

func TestReadToolsListStopsAtCompleteSSEEvent(t *testing.T) {
	payload := []byte("event: message\ndata: {\"result\":{\"tools\":[{\"name\":\"query-docs\"}]}}\n\n")
	got, err := readToolsList(&tailBlockingReader{payload: payload}, "text/event-stream", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("query-docs")) {
		t.Fatalf("tools/list 事件未返回: %q", got)
	}
}

func TestReadToolsListReadsJSONToEOF(t *testing.T) {
	got, err := readToolsList(bytes.NewBufferString(`{"result":{"tools":[{"name":"query-docs"}]}}`), "application/json", 1<<20)
	if err != nil || !bytes.Contains(got, []byte("query-docs")) {
		t.Fatalf("JSON tools/list 未读取: %q, %v", got, err)
	}
}
