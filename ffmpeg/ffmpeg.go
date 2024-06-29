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

func StartStreaming(source string, streamId string, backUp int) (cmd *exec.Cmd, err error) {
	cmd = exec.CommandContext(
		context.Background(),
		"ffmpeg",
		//"-version",
		"-stream_loop",
		"-1",
		"-i",
		source,
		"-b:v",
		"2500k",
		"-acodec",
		"libmp3lame",
		"-ar",
		"44100",
		"-threads",
		"6",
		"-qscale",
		"3",
		"-b:a",
		"712000",
		//"-vcodec",
		//"libx264",
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
