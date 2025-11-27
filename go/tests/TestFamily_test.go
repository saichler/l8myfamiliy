package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	m.Run()
	tear()
}

func TestFamily(t *testing.T) {
	exec.Command("rm", "-rf", "./web").Run()
	os.CopyFS("./web", os.DirFS("../myf/webui/web"))
	defer exec.Command("rm", "-rf", "./web").Run()
	startWebServer(9092, "test")
}
