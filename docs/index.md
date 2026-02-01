---
title: Home
---

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jovandeginste/workout-tracker)](https://github.com/jovandeginste/workout-tracker/blob/master/go.mod)
[![GitHub Release](https://img.shields.io/github/v/release/jovandeginste/workout-tracker)](https://github.com/jovandeginste/workout-tracker/releases/latest)
[![GitHub Downloads (all assets, latest release)](https://img.shields.io/github/downloads/jovandeginste/workout-tracker/latest/total)](https://github.com/jovandeginste/workout-tracker/releases/latest)

[![Go Report Card](https://goreportcard.com/badge/github.com/jovandeginste/workout-tracker)](https://goreportcard.com/report/github.com/jovandeginste/workout-tracker)
[![Libraries.io dependency status for GitHub repo](https://img.shields.io/librariesio/github/jovandeginste/workout-tracker)](https://libraries.io/go/github.com%2Fjovandeginste%2Fworkout-tracker%2Fv2)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Swagger Validator](https://img.shields.io/swagger/valid/3.0?specUrl=https%3A%2F%2Fraw.githubusercontent.com%2Fjovandeginste%2Fworkout-tracker%2Fmaster%2Fdocs%2Fswagger.json)](https://editor.swagger.io/?url=https://raw.githubusercontent.com/jovandeginste/workout-tracker/master/docs/swagger.json)

[![Translation status](https://hosted.weblate.org/widget/workout-tracker/svg-badge.svg)](https://hosted.weblate.org/engage/workout-tracker/)

[![Chat on Matrix](https://matrix.to/img/matrix-badge.svg)](https://matrix.to/#/#workout-tracker:matrix.org)

A workout tracking web application for personal use (or family, friends), geared
towards running and other GPX-based activities

Self-hosted, everything included.

Chat with the community
[on Matrix](https://matrix.to/#/#workout-tracker:matrix.org)

## Features

- upload workout records (gpx, tcx or fit files)
  - manually, or automatically via API (eg. Fitotrack)
- keep track of personal daily stats (weight, step count, ...)
  - manually, or via API and sync (eg. [Fitbit](./cmd/fitbit-sync/))
- create manual workout records (weight lifting, push-ups, swimming, ...)
- create route segments to keep track of your progress
  - this application will try to detect matches of your workouts
- keep track of equipment you are using
- check your progress through statistics
- see your "heatmap": where have you been (a lot)?

## :heart: Donate your workout files :heart:

We are collecting real workout files for testing purposes. If you want to
support the project, donate your files. You can open an issue and attach the
file, or send a pull request.

We are looking for general files from all sources, but also "raw" files from
devices. The first file type can be edited to remove personal information, but
the second type should be as pure (raw) as possible.

Make sure the file does not contain personally identifiable information,
specifically your home address! Preferably, share files from workouts that you
recorded while travelling.

Be sure to add some metadata in the issue or pull request, such as:

- the activity type (running, swimming, ...)
- the general location's name (city, national park, ...)
- anything relevant, such as whether there is heart rate or cadence data, or any
  other data

By donating the files, you grant the project full permission to use them as they
see fit.

## Compatiblity

This is a work in progress. If you find any problems, please let us know. The
application is tested with GPX files from these sources:

- Garmin Connect (export to GPX)
- FitoTrack (automatic export to GPX)
- Workoutdoors (export to GPX)
- Runtastic / Adidas Running using
  [this tool](https://github.com/Griffsano/RuntasticConverter)
