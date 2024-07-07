package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

func CheckInstall() {
	command := exec.Command(
		"ffmpeg",
		"-version",
	)

	var out bytes.Buffer
	var errorOut bytes.Buffer
	command.Stdout = &out
	command.Stderr = &errorOut
	err := command.Run()

	if err != nil {
		panic("require ffmpeg")
	}
}

func ConvertCodec(source string, destination string) (cmd *exec.Cmd, err error) {
	cmd = exec.CommandContext(
		context.Background(),
		"ffmpeg",
		"-i",
		source,
		"-c:v",
		"libx264",
		"-preset",
		"slow",
		"-crf",
		"22",
		"-c:a",
		"copy",
		destination,
	)

	var out bytes.Buffer
	var errorOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errorOut
	err = cmd.Run()

	return
}

func StartStreaming(source string, streamId string, backUp int) (cmd *exec.Cmd, err error) {
	cmd = exec.CommandContext(
		context.Background(),
		"ffmpeg",
		"-re",
		"-stream_loop",
		"-1",
		"-i",
		source,
		//"-filter:v", "fps=30",
		//"-c:v", "libx264",
		"-qscale", "3", "-b:a", "712000",
		"-f",
		"flv",
		fmt.Sprintf("rtmp://b.rtmp.youtube.com/live2/%s?backup=%d", streamId, backUp),
	)

	var out bytes.Buffer
	var errorOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errorOut
	err = cmd.Start()
	return
}
