package main

import "testing"

func TestShouldNotifyOnlyReadyAssets(t *testing.T) {
	tests := []struct {
		name  string
		event AssetEvent
		want  bool
	}{
		{"ready", AssetEvent{AssetID: "a1", Creator: "c1", State: "ready"}, true},
		{"processing", AssetEvent{AssetID: "a1", Creator: "c1", State: "processing"}, false},
		{"missing creator", AssetEvent{AssetID: "a1", State: "ready"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldNotify(tt.event); got != tt.want {
				t.Fatalf("shouldNotify() = %v, want %v", got, tt.want)
			}
		})
	}
}
