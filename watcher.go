package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"validator/config"
)

type (
	RemoteError struct {
		Msg string
	}

	Cfg struct {
		Logread   string `json:"logreader"`
		Pattern   string `json:"pattern"`
		SSHPath   string `json:"ssh_path"`
		SSHPort   string `json:"ssh_port"`
		SSHUrl    string `json:"ssh_url"`
		RSAPath   string `json:"rsa_path"`
		Validator string `json:"validator"`
	}
)

var cfg *Cfg

func (e *RemoteError) Error() string {
	return fmt.Sprintf("RemoteError: %s", e.Msg)
}

func validateRemote(http string) error {

	if idx := strings.Index(http, cfg.Pattern); idx != -1 {
		http = http[idx:]
	} else {
		return &RemoteError{
			Msg: "Index not found!",
		}
	}

	cmd := exec.Command(cfg.SSHPath,
		"-i", cfg.RSAPath,
		"-p", cfg.SSHPort,
		cfg.SSHUrl,
		cfg.Validator, `"`+http+`"`)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "stdout:", stdout.String())
		fmt.Fprintln(os.Stderr, "stderr:", stderr.String())
		return err
	}

	if stderr.Len() > 0 {
		return &RemoteError{
			Msg: stderr.String(),
		}
	}

	if stdout.Len() > 0 {
		outstr := stdout.String()
		if strings.Contains(outstr, "Approve ok!") {
			return nil
		}
		return &RemoteError{
			Msg: outstr,
		}
	}

	return &RemoteError{
		Msg: "Unknown Error",
	}
}

func writelastdetected(line string) error {
	file, err := os.Create("lastdetected.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(line)
	return err
}

func lastdetected(line string) bool {

	file, err := os.Open("lastdetected.txt")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(line, scanner.Text()) {
			return true
		}
	}

	return false
}

func detect(lines []string) {
	size := len(lines) - 1

	for i := size; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, cfg.Pattern) {

			if !lastdetected(line) {
				fmt.Fprintln(os.Stdout, "Approve required.")
				if err := validateRemote(line); err == nil {
					writelastdetected(line)
				} else {
					fmt.Fprintln(os.Stderr, err)
				}
				return
			}

			break
		}
	}

	fmt.Fprintln(os.Stdout, "No approve required.")
}

func main() {

	cfg = &Cfg{}

	if err := config.Load("watcher.json", cfg); err != nil {
		cfg = &Cfg{
			Logread:   "logread",
			Pattern:   "https://APPROVE-PATTERN-DOMAIN/cdn-cgi/access/cli?aud=",
			SSHPath:   "/usr/bin/ssh",
			SSHPort:   "22",
			SSHUrl:    "USER@VALIDATOR.SSH.DOMAIN",
			RSAPath:   "PATH/TO/RSASSH-KEY",
			Validator: "validator",
		}
		if err := config.Save("watcher.json", cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
	}

	cmd := exec.Command(cfg.Logread)
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Stdoutpipe %v\n", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Start logread: %v\n", err)
		os.Exit(1)
	}

	var lines []string
	donechan := make(chan bool)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Scanner err: %v\n", err)
		}
		donechan <- true
	}()

	timeout := 30 * time.Second
	select {
	case <-time.After(timeout):
		fmt.Fprintln(os.Stderr, "Process timeout.")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		time.Sleep(100 * time.Microsecond)
	case <-donechan:
		detect(lines)
	}

}
