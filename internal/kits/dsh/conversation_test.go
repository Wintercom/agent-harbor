package dsh

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestConversationBuildUsesFixedDriverAndFramedInput(t *testing.T) {
	k := fixtureKit(t)
	k.conversation = true
	input := validInput()
	launch, err := k.Build(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !launch.Interactive || launch.EncodeInput == nil || len(launch.ConfigFiles) != 4 {
		t.Fatalf("%+v", launch)
	}
	encoded, err := launch.EncodeInput([]byte("round\n\"two\""))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(encoded), "\n") != 1 {
		t.Fatal("正文注入额外协议消息")
	}
	var message struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(encoded, &message); err != nil || message.Text != "round\n\"two\"" {
		t.Fatal("输入编码错误", err)
	}
	source := string(launch.ConfigFiles[3].Content)
	if strings.Contains(source, "__AGENT_MODULE__") || strings.Contains(source, input.Task) {
		t.Fatal("驱动未解析或混入任务")
	}
	for _, bad := range [][]byte{nil, []byte(" "), {0xff}, {0}, []byte(strings.Repeat("a", 32769))} {
		if _, err := launch.EncodeInput(bad); err == nil {
			t.Fatal("接受非法输入")
		}
	}
}
