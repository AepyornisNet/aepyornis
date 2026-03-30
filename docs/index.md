---
title: Home
---

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/AepyornisNet/aepyornis)](https://github.com/AepyornisNet/aepyornis/blob/main/go.mod)
[![GitHub Release](https://img.shields.io/github/v/release/AepyornisNet/aepyornis)](https://github.com/AepyornisNet/aepyornis/releases/latest)
[![GitHub Downloads (all assets, latest release)](https://img.shields.io/github/downloads/AepyornisNet/aepyornis/latest/total)](https://github.com/AepyornisNet/aepyornis/releases/latest)

[![Go Report Card](https://goreportcard.com/badge/github.com/AepyornisNet/aepyornis)](https://goreportcard.com/report/github.com/AepyornisNet/aepyornis)
[![Libraries.io dependency status for GitHub repo](https://img.shields.io/librariesio/github/AepyornisNet/aepyornis)](https://libraries.io/go/github.com%2FAepyornisNet%2Faepyornis%2Fv2)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Swagger Validator](https://img.shields.io/swagger/valid/3.0?specUrl=https%3A%2F%2Fraw.githubusercontent.com%2FAepyornisNet%2Faepyornis%2Fmain%2Fdocs%2Fswagger.json)](https://editor.swagger.io/?url=https://raw.githubusercontent.com/AepyornisNet/aepyornis/main/docs/swagger.json)

A workout tracking web application for personal use (or family, friends), geared
towards running and other GPX-based activities

Self-hosted, everything included.

## Features

- upload workout records (gpx, tcx or fit files)
  - manually, or automatically via API (eg. Fitotrack)
- keep track of personal daily stats (weight, step count, ...)
- create manual workout records (weight lifting, push-ups, swimming, ...)
- create route segments to keep track of your progress
  - this application will try to detect matches of your workouts
- keep track of equipment you are using
- check your progress through statistics
- see your "heatmap": where have you been (a lot)?
- federate with other Aepyornis instances via [ActivityPub](./acitivitypub.html)
