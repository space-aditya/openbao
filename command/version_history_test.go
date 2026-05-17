// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/cli"
	"github.com/openbao/openbao/version"
)

func testVersionHistoryCommand(tb testing.TB) (*cli.MockUi, *VersionHistoryCommand) {
	tb.Helper()

	ui := cli.NewMockUi()
	return ui, &VersionHistoryCommand{
		BaseCommand: &BaseCommand{
			UI: ui,
		},
	}
}

func TestVersionHistoryCommand_TableOutput(t *testing.T) {
	t.Parallel()

	client, closer := testVaultServer(t)
	defer closer()

	ui, cmd := testVersionHistoryCommand(t)
	cmd.client = client

	var code int
	// The version history recording is asynchronous on server startup.
	// Retry until the output contains the current version.
	for range 10 {
		code = cmd.Run([]string{})
		if code == 0 && strings.Contains(ui.OutputWriter.String(), version.Version) {
			break
		}
		ui.OutputWriter.Reset()
		ui.ErrorWriter.Reset()
		time.Sleep(100 * time.Millisecond)
	}

	if expectedCode := 0; code != expectedCode {
		t.Fatalf("expected %d to be %d: %s", code, expectedCode, ui.ErrorWriter.String())
	}

	if errorString := ui.ErrorWriter.String(); !strings.Contains(errorString, versionTrackingWarning) {
		t.Errorf("expected %q to contain %q", errorString, versionTrackingWarning)
	}

	output := ui.OutputWriter.String()

	if !strings.Contains(output, version.Version) {
		t.Errorf("expected %q to contain version %q", output, version.Version)
	}
}

func TestVersionHistoryCommand_JsonOutput(t *testing.T) {
	t.Parallel()

	client, closer := testVaultServer(t)
	defer closer()

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	runOpts := &RunOptions{
		Stdout: stdout,
		Stderr: stderr,
		Client: client,
	}

	args, format, _, _, _ := setupEnv([]string{"version-history", "-format", "json"})
	if format != "json" {
		t.Fatalf("expected format to be %q, actual %q", "json", format)
	}

	var code int
	var stdoutBytes []byte
	// The version history recording is asynchronous on server startup.
	for range 10 {
		stdout.Reset()
		stderr.Reset()
		code = RunCustom(args, runOpts)
		stdoutBytes = stdout.Bytes()
		if code == 0 && json.Valid(stdoutBytes) && strings.Contains(string(stdoutBytes), version.Version) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if expectedCode := 0; code != expectedCode {
		t.Fatalf("expected %d to be %d: %s", code, expectedCode, stderr.String())
	}

	if stderrString := stderr.String(); !strings.Contains(stderrString, versionTrackingWarning) {
		t.Errorf("expected %q to contain %q", stderrString, versionTrackingWarning)
	}

	if !json.Valid(stdoutBytes) {
		t.Fatalf("expected output %q to be valid JSON", stdoutBytes)
	}

	var versionHistoryResp map[string]interface{}
	err := json.Unmarshal(stdoutBytes, &versionHistoryResp)
	if err != nil {
		t.Fatalf("failed to unmarshal json from STDOUT, err: %s", err.Error())
	}

	var respData map[string]interface{}
	var ok bool
	var keys []interface{}
	var keyInfo map[string]interface{}

	if respData, ok = versionHistoryResp["data"].(map[string]interface{}); !ok {
		t.Fatalf("expected data key to be map, actual: %#v", versionHistoryResp["data"])
	}

	if keys, ok = respData["keys"].([]interface{}); !ok {
		t.Fatalf("expected keys to be array, actual: %#v", respData["keys"])
	}

	if keyInfo, ok = respData["key_info"].(map[string]interface{}); !ok {
		t.Fatalf("expected key_info to be map, actual: %#v", respData["key_info"])
	}

	if len(keys) != 1 {
		t.Fatalf("expected single version history entry for %q", version.Version)
	}

	if keyInfo[version.Version] == nil {
		t.Fatalf("expected version %s to be present in key_info, actual: %#v", version.Version, keyInfo)
	}
}
