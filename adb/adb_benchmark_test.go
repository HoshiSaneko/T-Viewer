package adb

import (
	"strings"
	"testing"
)

var benchmarkXML = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="" class="android.widget.FrameLayout" bounds="[0,0][1080,2400]">
    <node index="0" text="Test Node 1" class="android.widget.TextView" bounds="[100,100][200,200]">
		<node index="0" text="Test Node 1.1" class="android.widget.TextView" bounds="[100,100][200,200]" />
		<node index="1" text="Test Node 1.2" class="android.widget.TextView" bounds="[100,100][200,200]" />
	</node>
    <node index="1" text="Test Node 2" class="android.widget.TextView" bounds="[100,100][200,200]" />
    <node index="2" text="Test Node 3" class="android.widget.TextView" bounds="[100,100][200,200]" />
    <node index="3" text="Test Node 4" class="android.widget.TextView" bounds="[100,100][200,200]" />
  </node>
</hierarchy>`

func BenchmarkParseDevices(b *testing.B) {
	output := `List of devices attached
device123      device usb:123456X product:Pixel_4 model:Pixel_4 device:flame
device456      offline usb:123456Y
device789      unauthorized
`
	for i := 0; i < b.N; i++ {
		ParseDevices(output)
	}
}

func BenchmarkParseUIHierarchy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(benchmarkXML)
		ParseUIHierarchy(reader)
	}
}
