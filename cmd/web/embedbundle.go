package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// connStatusBundle holds pre-rendered CSS/JS for embed surfaces.
type connStatusBundle struct {
	css              string
	inlineJS         string
	inlineDetailsJS  string
	embedScript      []byte
}

func loadConnStatusBundle(probeV4, probeV6, probeDS, siteURL string) (*connStatusBundle, error) {
	cssBytes, err := staticFS.ReadFile("static/css/conn-status-embed.css")
	if err != nil {
		return nil, fmt.Errorf("conn-status css: %w", err)
	}
	probeJS, err := buildProbeBootstrap(probeV4, probeV6, probeDS)
	if err != nil {
		return nil, err
	}
	probeFile, err := staticFS.ReadFile("static/js/conn-status-probe.js")
	if err != nil {
		return nil, fmt.Errorf("conn-status-probe.js: %w", err)
	}
	uiFile, err := staticFS.ReadFile("static/js/conn-status-ui.js")
	if err != nil {
		return nil, fmt.Errorf("conn-status-ui.js: %w", err)
	}
	runFile, err := staticFS.ReadFile("static/js/conn-status-run.js")
	if err != nil {
		return nil, fmt.Errorf("conn-status-run.js: %w", err)
	}
	detailsRunFile, err := staticFS.ReadFile("static/js/conn-status-details-run.js")
	if err != nil {
		return nil, fmt.Errorf("conn-status-details-run.js: %w", err)
	}
	mountFile, err := staticFS.ReadFile("static/js/conn-status-embed-mount.js")
	if err != nil {
		return nil, fmt.Errorf("conn-status-embed-mount.js: %w", err)
	}

	inlineJS := probeJS + "\n" + string(probeFile) + "\n" + string(uiFile) + "\n" + string(runFile)
	inlineDetailsJS := probeJS + "\n" + string(probeFile) + "\n" + string(uiFile) + "\n" + string(detailsRunFile)

	mountJS := strings.ReplaceAll(string(mountFile), "{{SITE_URL}}", siteURL)
	var embedBuf bytes.Buffer
	embedBuf.WriteString(probeJS)
	embedBuf.WriteByte('\n')
	embedBuf.Write(probeFile)
	embedBuf.WriteByte('\n')
	embedBuf.Write(uiFile)
	embedBuf.WriteByte('\n')
	embedBuf.WriteString(mountJS)

	return &connStatusBundle{
		css:             string(cssBytes),
		inlineJS:        inlineJS,
		inlineDetailsJS: inlineDetailsJS,
		embedScript:     embedBuf.Bytes(),
	}, nil
}

func buildProbeBootstrap(probeV4, probeV6, probeDS string) (string, error) {
	v4, err := json.Marshal(probeV4)
	if err != nil {
		return "", err
	}
	v6, err := json.Marshal(probeV6)
	if err != nil {
		return "", err
	}
	ds, err := json.Marshal(probeDS)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"window.__PROBE_V4__=%s;window.__PROBE_V6__=%s;window.__PROBE_DS__=%s;",
		v4, v6, ds,
	), nil
}
